package icecast

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"server/src/db"
	"server/src/session"
	"time"
)

const (
	ticksPerSecond = 50
)

type Mountpoint string

type IcecastClient struct {
	sessionManager     *session.SessionManager
	icecastHost        string
	icecastPort        string
	sourcePassword     string
	silenceTrackBuffer []byte
}

func CreateIcecastClient(
	sessionManager *session.SessionManager,
	icecastHost string,
	icecastPort string,
	sourcePassword string,
	silenceTrackPath string,
) (*IcecastClient, error) {
	silenceTrackBuffer, err := makeSilenceBuffer(silenceTrackPath)
	if err != nil {
		return nil, err
	}

	return &IcecastClient{
		sessionManager:     sessionManager,
		icecastHost:        icecastHost,
		icecastPort:        icecastPort,
		sourcePassword:     sourcePassword,
		silenceTrackBuffer: silenceTrackBuffer,
	}, nil
}

func (i *IcecastClient) StreamSessions(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-i.sessionManager.Events:
			if !ok {
				return
			}
			switch event.Type {
			case session.SessionCreated:
				queue := i.sessionManager.GetQueue(event.SessionID)
				if queue != nil {
					go i.streamSession(ctx, queue, event.SessionID)
				}
			}
		}
	}
}

func (i *IcecastClient) streamSession(ctx context.Context, queue *session.SessionQueue, sessionID session.SessionID) {
	mountpoint := Mountpoint(sessionID)
	conn, err := i.icecastConnection(ctx, mountpoint)
	if err != nil {
		slog.Error("failed to connect to icecast", "mountpoint", mountpoint, "err", err)
		return
	}
	defer conn.Close()
	slog.Info("stream started", "mountpoint", mountpoint)

	silenceTicker := time.NewTicker(time.Second / ticksPerSecond)
	defer silenceTicker.Stop()

	sessionEndTimer := time.NewTimer(15 * time.Minute)
	defer sessionEndTimer.Stop()

	for {
		for {
			track, err := queue.Dequeue()
			if errors.Is(err, session.EmptyQueueError) {
				break
			}
			if err != nil {
				return
			}
			sessionEndTimer.Reset(10 * time.Minute)
			slog.Info("playing track", "title", track.Title, "artist", track.Artist, "mountpoint", mountpoint)
			silenceTicker.Stop()
			if err := i.streamTrack(ctx, track, conn); err != nil {
				slog.Error("stream error", "mountpoint", mountpoint, "err", err)
				return
			}
		}
		silenceTicker.Reset(time.Second / ticksPerSecond)

		select {
		case <-ctx.Done():
			slog.Info("stream cancelled", "mountpoint", mountpoint)
			return
		case <-queue.Notify():
			// loop back to drain
		case <-silenceTicker.C:
			if err := i.writeSilenceChunk(conn); err != nil {
				return
			}
		case <-sessionEndTimer.C:
			slog.Info("closing stream after silence timeout", "mountpoint", mountpoint)
			i.sessionManager.DeleteSession(ctx, sessionID)
			return
		}
	}
}

func (i *IcecastClient) streamTrack(ctx context.Context, track db.Track, conn io.Writer) error {
	f, err := os.Open(track.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	chunkSize := max(info.Size()/(track.Duration*ticksPerSecond), 1)
	buf := make([]byte, chunkSize)

	ticker := time.NewTicker(time.Second / ticksPerSecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			n, err := f.Read(buf)
			if n > 0 {
				if _, werr := conn.Write(buf[:n]); werr != nil {
					return werr
				}
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

func (i *IcecastClient) writeSilenceChunk(conn io.Writer) error {
	_, err := conn.Write(i.silenceTrackBuffer)
	return err
}

func (i *IcecastClient) icecastConnection(ctx context.Context, mountpoint Mountpoint) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	url := fmt.Sprintf("http://%s:%s/%s", i.icecastHost, i.icecastPort, mountpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "audio/ogg")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("source:"+i.sourcePassword)))

	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		resp.Body.Close()
	}()

	return pw, nil
}

func makeSilenceBuffer(silenceTrackPath string) ([]byte, error) {
	f, err := os.Open(silenceTrackPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, 4*1024)
	n, err := f.Read(buf)
	if n > 0 {
		return buf, nil
	}
	return nil, err
}
