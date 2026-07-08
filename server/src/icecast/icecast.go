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
	"path"
	"server/src/db"
	"server/src/session"
	"strings"
	"sync"
	"time"

	"server/src/util"
)

const (
	SESSION_TIMEOUT_MINUTES = 60
	OPUS_BITRATE            = "128k"
	STREAM_PATH_PREFIX      = "stream"
	PCM_SAMPLE_RATE         = "48000"
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
	mu              sync.Mutex
	cancels         map[session.SessionID]context.CancelFunc
	db              *db.DB
}

func CreateIcecastClient(
	sessionManager *session.SessionManager,
	icecastHost string,
	icecastPort string,
	icecastPassword string,
	streamBaseURL string,
	d *db.DB,
) *IcecastClient {
	return &IcecastClient{
		sessionManager:  sessionManager,
		icecastHost:     icecastHost,
		icecastPort:     icecastPort,
		icecastPassword: icecastPassword,
		streamBaseURL:   streamBaseURL,
		cancels:         make(map[session.SessionID]context.CancelFunc),
		db:              d,
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
				queue, err := i.sessionManager.GetQueue(event.SessionID)
				if err != nil {
					slog.Error("icecast: session not found after creation event", "sessionID", event.SessionID, "err", err)
					return
				}
				sessionCtx, cancel := context.WithCancel(ctx)
				i.mu.Lock()
				i.cancels[event.SessionID] = cancel
				i.mu.Unlock()
				go i.streamSession(sessionCtx, queue, event.SessionID, event.Ready)
			case session.SessionDeleted:
				i.mu.Lock()
				cancel, ok := i.cancels[event.SessionID]
				delete(i.cancels, event.SessionID)
				i.mu.Unlock()
				if ok {
					slog.Info("icecast ending stream for deleted session", "sessionID", event.SessionID)
					cancel()
				}
			}
		}
	}
}

func (i *IcecastClient) streamSession(ctx context.Context, queue *session.SessionQueue, sessionID session.SessionID, ready chan error) {
	mountpoint := Mountpoint(STREAM_PATH_PREFIX + "/" + string(sessionID))

	endSession := func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := i.sessionManager.DeleteSession(deleteCtx, sessionID); err != nil {
			slog.Error("failed to delete session", "mountpoint", mountpoint, "err", err)
		}
	}
	defer endSession()

	icecastConnection, err := i.connectWithRetry(ctx, mountpoint)
	if err != nil {
		slog.Error("failed to connect to icecast", "mountpoint", mountpoint, "err", err)
		ready <- err
		return
	}
	defer icecastConnection.Close()
	slog.Info("stream started", "mountpoint", mountpoint)
	ready <- nil

	pcm, err := startEncoder(ctx, icecastConnection)
	if err != nil {
		slog.Error("failed to start encoder", "mountpoint", mountpoint, "err", err)
		return
	}
	defer pcm.Close()

	for {
		for {
			track, err := queue.Peek()
			if errors.Is(err, session.EmptyQueueError) {
				break
			}
			if err != nil {
				slog.Error("error peeking queue", "err", err, "mountpoint", mountpoint)
				return
			}

			slog.Info("playing track", "title", track.Title, "artist", track.Artist, "mountpoint", mountpoint)

			if archiveID := queue.ArchiveID(); archiveID != nil {
				if err := i.db.AddSessionArchiveTrack(ctx, *archiveID, track.Id); err != nil {
					slog.Error("failed to record archive track", "err", err, "mountpoint", mountpoint)
				}
			}

			var elapsed int64
			skipped := false
			for {
				trackCtx, stopSkipWatch := watchForSkip(ctx, queue)
				elapsed, err = streamTrackPCM(trackCtx, track, pcm, elapsed)
				skipped = stopSkipWatch()

				if err == nil || skipped {
					break
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				slog.Error("track decode error", "mountpoint", mountpoint, "err", err)
				return
			}

			if _, err := queue.Dequeue(); err != nil {
				slog.Error("error dequeuing track", "err", err, "mountpoint", mountpoint)
				return
			}
		}

		slog.Info("queue empty, streaming silence", "mountpoint", mountpoint)
		switch streamSilencePCM(ctx, pcm, queue) {
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

func startEncoder(ctx context.Context, w io.Writer) (io.WriteCloser, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "error",
		"-f", "s16le", "-ar", PCM_SAMPLE_RATE, "-ac", "2",
		"-i", "pipe:0",
		"-c:a", "libopus", "-vbr", "off", "-b:a", OPUS_BITRATE,
		"-f", "ogg",
		"pipe:1",
	)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	pcm, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go cmd.Wait()
	return pcm, nil
}

func streamTrackPCM(ctx context.Context, track *db.Track, w io.Writer, elapsed int64) (int64, error) {
	start := time.Now()
	args := []string{"-loglevel", "error", "-re"}
	if elapsed > 0 {
		args = append(args, "-ss", fmt.Sprintf("%d", elapsed))
	}
	args = append(args, "-i", track.FilePath, "-f", "s16le", "-ar", PCM_SAMPLE_RATE, "-ac", "2", "pipe:1")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		newElapsed := elapsed + int64(time.Since(start).Seconds())
		if ctx.Err() != nil {
			return newElapsed, ctx.Err()
		}
		return newElapsed, err
	}
	return 0, nil
}

type silenceResult int

const (
	silenceNewTrack  silenceResult = iota
	silenceTimedOut  silenceResult = iota
	silenceCancelled silenceResult = iota
)

func streamSilencePCM(ctx context.Context, w io.Writer, queue *session.SessionQueue) silenceResult {
	sessionEndTimer := time.NewTimer(SESSION_TIMEOUT_MINUTES * time.Minute)
	defer sessionEndTimer.Stop()

	silenceCtx, cancelSilence := context.WithCancel(ctx)
	silenceDone := make(chan struct{})
	go func() {
		defer close(silenceDone)
		cmd := exec.CommandContext(silenceCtx, "ffmpeg",
			"-loglevel", "error",
			"-re",
			"-f", "lavfi",
			"-i", "anullsrc=r="+PCM_SAMPLE_RATE+":cl=stereo",
			"-f", "s16le", "-ar", PCM_SAMPLE_RATE, "-ac", "2",
			"pipe:1",
		)
		cmd.Stdout = w
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && silenceCtx.Err() == nil {
			slog.Error("silence PCM stream error", "err", err)
		}
	}()

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

	stop := func() bool {
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

	return trackCtx, stop
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
