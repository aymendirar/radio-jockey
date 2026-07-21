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
	SessionTimeoutMinutes = 60
	OpusBitrate          = "128k"
	StreamPathPrefix     = "stream"
	PCMSampleRate        = "48000"

	sessionDeleteTimeout      = 5 * time.Second
	icecastResponseBufferSize = 512
	icecastReadTimeout        = 10 * time.Second
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
	database *db.DB,
) *IcecastClient {
	return &IcecastClient{
		sessionManager:  sessionManager,
		icecastHost:     icecastHost,
		icecastPort:     icecastPort,
		icecastPassword: icecastPassword,
		streamBaseURL:   streamBaseURL,
		cancels:         make(map[session.SessionID]context.CancelFunc),
		db:              database,
	}
}

func (i *IcecastClient) StreamURL(sessionID session.SessionID) string {
	return i.streamBaseURL + "/" + path.Join(StreamPathPrefix, string(sessionID))
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

// streamSession manages the full lifecycle of an Icecast stream for a single session:
// connect to Icecast, start the Opus encoder, play tracks as they arrive in the queue,
// and stream silence when the queue is empty. The stream ends when the context is
// cancelled or the session times out.
func (i *IcecastClient) streamSession(ctx context.Context, queue *session.SessionQueue, sessionID session.SessionID, ready chan error) {
	mountpoint := Mountpoint(StreamPathPrefix + "/" + string(sessionID))

	endSession := func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), sessionDeleteTimeout)
		defer cancel()
		if err := i.sessionManager.DeleteSession(deleteCtx, sessionID); err != nil {
			slog.Error("failed to delete session", "mountpoint", mountpoint, "err", err)
		}
	}
	defer endSession()

	icecastConn, err := i.connectWithRetry(ctx, mountpoint)
	if err != nil {
		slog.Error("failed to connect to icecast", "mountpoint", mountpoint, "err", err)
		ready <- err
		return
	}
	defer icecastConn.Close()
	slog.Info("stream started", "mountpoint", mountpoint)
	ready <- nil

	pcm, err := startEncoder(ctx, icecastConn)
	if err != nil {
		slog.Error("failed to start encoder", "mountpoint", mountpoint, "err", err)
		return
	}
	defer pcm.Close()

	for {
		for {
			cont, err := i.playCurrentTrack(ctx, queue, pcm, mountpoint)
			if err != nil {
				return
			}
			if !cont {
				break
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

// playCurrentTrack peeks at the next track in the queue, streams it to the encoder,
// and dequeues it on completion. Returns (true, nil) when a track was played (caller
// should continue), (false, nil) when the queue is empty (caller should stream silence),
// or (false, err) on a fatal error.
func (i *IcecastClient) playCurrentTrack(ctx context.Context, queue *session.SessionQueue, pcm io.Writer, mountpoint Mountpoint) (bool, error) {
	track, err := queue.Peek()
	if errors.Is(err, session.EmptyQueueError) {
		return false, nil
	}
	if err != nil {
		slog.Error("error peeking queue", "err", err, "mountpoint", mountpoint)
		return false, err
	}

	slog.Info("playing track", "title", track.Title, "artist", track.Artist, "mountpoint", mountpoint)

	if archiveID := queue.ArchiveID(); archiveID != nil {
		if err := i.db.AddSessionArchiveTrack(ctx, *archiveID, track.ID); err != nil {
			slog.Error("failed to record archive track", "err", err, "mountpoint", mountpoint)
		}
	}

	var elapsed int64
	for {
		trackCtx, stopSkipWatch := watchForSkip(ctx, queue)
		elapsed, err = streamTrackPCM(trackCtx, track, pcm, elapsed)
		skipped := stopSkipWatch()

		if err == nil || skipped {
			break
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, nil
		}
		slog.Error("track decode error", "mountpoint", mountpoint, "err", err)
		return false, err
	}

	if _, err := queue.Dequeue(); err != nil {
		slog.Error("error dequeuing track", "err", err, "mountpoint", mountpoint)
		return false, err
	}
	return true, nil
}

func startEncoder(ctx context.Context, w io.Writer) (io.WriteCloser, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "error",
		"-f", "s16le", "-ar", PCMSampleRate, "-ac", "2",
		"-i", "pipe:0",
		"-c:a", "libopus", "-vbr", "off", "-b:a", OpusBitrate,
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
	args = append(args, "-i", track.FilePath, "-f", "s16le", "-ar", PCMSampleRate, "-ac", "2", "pipe:1")
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
	silenceTimedOut
	silenceCancelled
)

func streamSilencePCM(ctx context.Context, w io.Writer, queue *session.SessionQueue) silenceResult {
	sessionEndTimer := time.NewTimer(SessionTimeoutMinutes * time.Minute)
	defer sessionEndTimer.Stop()

	silenceCtx, cancelSilence := context.WithCancel(ctx)
	silenceDone := make(chan struct{})
	go func() {
		defer close(silenceDone)
		cmd := exec.CommandContext(silenceCtx, "ffmpeg",
			"-loglevel", "error",
			"-re",
			"-f", "lavfi",
			"-i", "anullsrc=r="+PCMSampleRate+":cl=stereo",
			"-f", "s16le", "-ar", PCMSampleRate, "-ac", "2",
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

// watchForSkip monitors the queue's event channel for a skip signal while a track is
// playing. It returns a derived context (cancelled on skip) and a stop function. Calling
// stop cancels the goroutine, waits for it to finish, and returns true if a skip was
// received, false otherwise.
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

// icecastConnection opens a raw TCP connection to Icecast and performs the HTTP PUT
// handshake to start streaming a new mountpoint. Returns the connection for writing
// audio data, or an error if Icecast rejects the request.
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

	buf := make([]byte, icecastResponseBufferSize)
	conn.SetReadDeadline(time.Now().Add(icecastReadTimeout))
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
