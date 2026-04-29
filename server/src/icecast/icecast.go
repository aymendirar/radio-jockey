package icecast

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"server/src/db"
	"server/src/session"
	"strings"
	"time"

	"server/src/util"
)

const (
	TICKS_PER_SECOND = 50
	SESSION_TIMEOUT  = 10
	KILOBYTE         = 1024
)

type Mountpoint string

type IcecastClient struct {
	sessionManager     *session.SessionManager
	icecastHost        string
	icecastPort        string
	icecastPassword    string
	silenceTrackBuffer []byte
}

func CreateIcecastClient(
	sessionManager *session.SessionManager,
	icecastHost string,
	icecastPort string,
	icecastPassword string,
	silenceTrackPath string,
) (*IcecastClient, error) {
	silenceTrackBuffer, err := makeSilenceBuffer(silenceTrackPath)
	if err != nil {
		return nil, err
	}

	slog.Info("created icecast client")
	return &IcecastClient{
		sessionManager:     sessionManager,
		icecastHost:        icecastHost,
		icecastPort:        icecastPort,
		icecastPassword:    icecastPassword,
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
				slog.Info("icecast received new session, starting stream...", "sessionID", event.SessionID)
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

	conn, err := i.connectWithRetry(ctx, mountpoint)
	if err != nil {
		slog.Error("failed to connect to icecast", "mountpoint", mountpoint, "err", err)
		return
	}
	defer conn.Close()
	slog.Info("stream started", "mountpoint", mountpoint)

	reconnect := func() bool {
		conn.Close()
		conn, err = i.connectWithRetry(ctx, mountpoint)
		if err != nil {
			slog.Error("failed to reconnect to icecast", "mountpoint", mountpoint, "err", err)
			return false
		}
		slog.Info("reconnected to icecast", "mountpoint", mountpoint)
		return true
	}

	sessionEndTimer := time.NewTimer(SESSION_TIMEOUT * time.Minute)
	defer sessionEndTimer.Stop()

	sessionStart := time.Now()
	var totalDuration time.Duration

	for {
		for {
			track, err := queue.Dequeue()
			if errors.Is(err, session.EmptyQueueError) {
				break
			}
			if err != nil {
				return
			}
			totalDuration += time.Duration(track.Duration) * time.Second
			sessionEndTimer.Reset(SESSION_TIMEOUT * time.Minute)
			slog.Info("playing track", "title", track.Title, "artist", track.Artist, "mountpoint", mountpoint)

			var offset int64
			for {
				offset, err = i.streamTrack(ctx, track, conn, offset)
				if err == nil {
					break
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				slog.Error("stream error", "mountpoint", mountpoint, "err", err)
				if !reconnect() {
					return
				}
			}
		}

		// Wait for VLC to consume all buffered audio before injecting silence.
		// Without this, silence bytes corrupt the Ogg stream while music is still playing.
		if wait := time.Until(sessionStart.Add(totalDuration)); wait > 0 {
			select {
			case <-ctx.Done():
				slog.Info("stream cancelled", "mountpoint", mountpoint)
				return
			case <-queue.Notify():
				continue
			case <-time.After(wait):
			case <-sessionEndTimer.C:
				slog.Info("closing stream after silence timeout", "mountpoint", mountpoint)
				i.sessionManager.DeleteSession(ctx, sessionID)
				return
			}
		}

		select {
		case <-ctx.Done():
			slog.Info("stream cancelled", "mountpoint", mountpoint)
			return
		case <-queue.Notify():
			// loop back to drain
		case <-time.After(time.Second / TICKS_PER_SECOND):
			slog.Info("writing silence", "mountpoint", mountpoint)
			if err := i.writeSilenceChunk(conn); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				slog.Error("silence write error", "mountpoint", mountpoint, "err", err)
				if !reconnect() {
					return
				}
			}
		case <-sessionEndTimer.C:
			slog.Info("closing stream after silence timeout", "mountpoint", mountpoint)
			i.sessionManager.DeleteSession(ctx, sessionID)
			return
		}
	}
}

func (i *IcecastClient) streamTrack(ctx context.Context, track *db.Track, conn io.Writer, startOffset int64) (int64, error) {
	f, err := os.Open(track.FilePath)
	if err != nil {
		return startOffset, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return startOffset, err
	}

	if startOffset > 0 {
		if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
			return startOffset, err
		}
	}

	chunkSize := info.Size() / track.Duration / TICKS_PER_SECOND
	buf := make([]byte, chunkSize)
	offset := startOffset

	ticker := time.NewTicker(time.Second / TICKS_PER_SECOND)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return offset, ctx.Err()
		case <-ticker.C:
			n, err := f.Read(buf)
			if n > 0 {
				if _, werr := conn.Write(buf[:n]); werr != nil {
					return offset, werr
				}
				offset += int64(n)
			}
			if err == io.EOF {
				return offset, nil
			}
			if err != nil {
				return offset, err
			}
		}
	}
}

func (i *IcecastClient) writeSilenceChunk(conn io.Writer) error {
	_, err := conn.Write(i.silenceTrackBuffer)
	return err
}

func (i *IcecastClient) connectWithRetry(ctx context.Context, mountpoint Mountpoint) (io.WriteCloser, error) {
	var conn io.WriteCloser
	err := util.RetryWithBackoff(ctx, func() error {
		var err error
		conn, err = i.icecastConnection(ctx, mountpoint)
		return err
	})
	return conn, err
}

func (i *IcecastClient) icecastConnection(ctx context.Context, mountpoint Mountpoint) (io.WriteCloser, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", i.icecastHost, i.icecastPort))
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte("source:" + i.icecastPassword))
	req := fmt.Sprintf(
		"PUT /%s HTTP/1.1\r\nContent-Type: audio/ogg\r\nAuthorization: Basic %s\r\n\r\n",
		mountpoint, auth,
	)
	if _, err := fmt.Fprint(conn, req); err != nil {
		conn.Close()
		return nil, err
	}

	// Read the response line to check for 200 OK.
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		return nil, err
	}
	resp := string(buf[:n])
	if !strings.Contains(resp, "200") {
		conn.Close()
		return nil, fmt.Errorf("icecast rejected connection: %s", strings.TrimSpace(resp))
	}

	return conn, nil
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
		return buf[:n], nil
	}
	return nil, err
}
