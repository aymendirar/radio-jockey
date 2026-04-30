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
	"os/exec"
	"server/src/db"
	"server/src/session"
	"path"
	"strings"
	"time"

	"server/src/util"
)

const (
	SESSION_TIMEOUT_MINUTES = 10
	SILENCE_BITRATE         = "128k"
	STREAM_PATH_PREFIX      = "stream"
)

var (
	ErrIcecastRejected = errors.New("icecast rejected connection")
)

type Mountpoint string

type IcecastClient struct {
	sessionManager  *session.SessionManager
	icecastHost     string
	icecastPort     string
	icecastPassword string
	streamBaseURL   string
}

func CreateIcecastClient(
	sessionManager *session.SessionManager,
	icecastHost string,
	icecastPort string,
	icecastPassword string,
	streamBaseURL string,
) *IcecastClient {
	return &IcecastClient{
		sessionManager:  sessionManager,
		icecastHost:     icecastHost,
		icecastPort:     icecastPort,
		icecastPassword: icecastPassword,
		streamBaseURL:   streamBaseURL,
	}
}

func (i *IcecastClient) StreamURL(sessionID session.SessionID) string {
	return i.streamBaseURL + "/" + path.Join(STREAM_PATH_PREFIX, string(sessionID))
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
	mountpoint := Mountpoint(STREAM_PATH_PREFIX + "/" + string(sessionID))

	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := i.sessionManager.DeleteSession(deleteCtx, sessionID); err != nil {
			slog.Error("failed to delete session", "mountpoint", mountpoint, "err", err)
		}
	}()

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

	for {
		for {
			track, err := queue.Dequeue()
			if errors.Is(err, session.EmptyQueueError) {
				break
			}
			if err != nil {
				slog.Error("error while dequeuing track", "err", err, "mountpoint", mountpoint)
				return
			}

			slog.Info("playing track", "title", track.Title, "artist", track.Artist, "mountpoint", mountpoint)

			var elapsedTrackTime int64
			for {
				trackCtx, stopSkipWatch := watchForSkip(ctx, queue)
				elapsedTrackTime, err = i.streamTrack(trackCtx, track, conn, elapsedTrackTime)
				skipped := stopSkipWatch()

				if err == nil || skipped {
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

		slog.Info("queue empty, streaming silence", "mountpoint", mountpoint)
		switch i.streamSilence(ctx, conn, queue) {
		case silenceCancelled:
			slog.Info("stream cancelled", "mountpoint", mountpoint)
			return
		case silenceTimedOut:
			slog.Info("session timed out", "mountpoint", mountpoint)
			return
		case silenceNewTrack:
			slog.Info("playing next track...", "mountpoint", mountpoint)
		}
	}
}

type silenceResult int

const (
	silenceNewTrack  silenceResult = iota
	silenceTimedOut  silenceResult = iota
	silenceCancelled silenceResult = iota
)

func (i *IcecastClient) streamSilence(ctx context.Context, conn io.Writer, queue *session.SessionQueue) silenceResult {
	sessionEndTimer := time.NewTimer(SESSION_TIMEOUT_MINUTES * time.Minute)
	defer sessionEndTimer.Stop()
	cancelSilence, silenceDone := i.startSilence(ctx, conn)

	var result silenceResult
	select {
	case <-ctx.Done():
		result = silenceCancelled
	case <-queue.Notify():
		result = silenceNewTrack
	case <-sessionEndTimer.C:
		result = silenceTimedOut
	}
	cancelSilence()
	<-silenceDone
	return result
}

func watchForSkip(ctx context.Context, queue *session.SessionQueue) (context.Context, func() bool) {
	trackCtx, cancel := context.WithCancel(ctx)
	skipped := false
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case event := <-queue.Events:
			if event.Type == session.SkipTrack {
				skipped = true
				cancel()
			}
		case <-trackCtx.Done():
		}
	}()
	return trackCtx, func() bool {
		cancel()
		<-done
		if !skipped {
			select {
			case <-queue.Events:
			default:
			}
		}
		return skipped
	}
}

func (i *IcecastClient) startSilence(ctx context.Context, conn io.Writer) (context.CancelFunc, <-chan struct{}) {
	silenceCtx, cancelSilence := context.WithCancel(ctx)
	silenceDone := make(chan struct{})
	go func() {
		defer close(silenceDone)
		i.streamSilenceLoop(silenceCtx, conn)
	}()
	return cancelSilence, silenceDone
}

func (i *IcecastClient) streamSilenceLoop(ctx context.Context, conn io.Writer) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "error",
		"-re",
		"-f", "lavfi",
		"-i", "anullsrc=r=48000:cl=stereo",
		"-c:a", "libopus", "-vbr", "off", "-b:a", SILENCE_BITRATE,
		"-f", "ogg",
		"pipe:1",
	)
	cmd.Stdout = conn
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && ctx.Err() == nil {
		slog.Error("silence stream error", "err", err)
	}
}

func (i *IcecastClient) streamTrack(ctx context.Context, track *db.Track, conn io.Writer, elapsedTrackTime int64) (int64, error) {
	start := time.Now()

	args := []string{"-loglevel", "error", "-re"}
	if elapsedTrackTime > 0 {
		args = append(args, "-ss", fmt.Sprintf("%d", elapsedTrackTime))
	}
	args = append(args, "-i", track.FilePath, "-c:a", "copy", "-f", "ogg", "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = conn
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		newElapsed := elapsedTrackTime + int64(time.Since(start).Seconds())
		if ctx.Err() != nil {
			return newElapsed, ctx.Err()
		}
		return newElapsed, err
	}
	return 0, nil
}

func (i *IcecastClient) connectWithRetry(ctx context.Context, mountpoint Mountpoint) (io.WriteCloser, error) {
	var conn io.WriteCloser
	err := util.RetryWithBackoff(
		ctx,
		func() error {
			var err error
			conn, err = i.icecastConnection(ctx, mountpoint)
			return err
		},
		func(n uint, err error) {
			slog.Warn("failed to connect to icecast server", "mountpoint", mountpoint)
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

	buf := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(buf)
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp := string(buf[:n])
	if !strings.Contains(resp, "200") {
		conn.Close()
		return nil, fmt.Errorf("%w: %s", ErrIcecastRejected, strings.TrimSpace(resp))
	}

	return conn, nil
}
