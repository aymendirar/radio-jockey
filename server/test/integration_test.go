package integration_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	serverconnect "server/src/connect"
	"server/src/db"
	"server/src/icecast"
	"server/src/music"
	"server/src/proto"
	"server/src/proto/protoconnect"
	"server/src/session"
	"server/test/util"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
)

// testServer boots the real HTTP server (real RPC handlers, real SessionManager, real
// IcecastClient) against a fake Icecast TCP stand-in, so tests exercise the actual
// production wiring end to end without mocking any Go code. The only thing not real is
// the Icecast *service* itself (and AddTrack's yt-dlp download, which individual tests
// route around by seeding tracks directly) — everything else, including ffmpeg encoding,
// runs for real.
type testServer struct {
	client         protoconnect.RadioServiceClient
	db             *db.DB
	sessionManager *session.SessionManager
	icecast        *util.FakeIcecastServer
	auth           *util.TestAuth
}

// setupTestServer wires up the real HTTP server, RPC handlers, SessionManager and
// IcecastClient against a fake Icecast TCP stand-in. When realAudio is true, the actual
// icecast.StreamSessions loop runs, so sessions immediately start consuming their queue in
// real time via real ffmpeg processes — use this only for the test that specifically
// exercises that pipeline. Otherwise a lightweight stand-in just acks session readiness
// without touching the queue, so other tests can inspect queue state deterministically.
// maxSessions, rateLimitRPS and rateLimitBurst are threaded straight through to the real
// SessionManager/server config; tests that don't care about those limits pass generous
// values so the caps never trip incidentally.
func setupTestServer(t *testing.T, realAudio bool, maxSessions int, rateLimitRPS float64, rateLimitBurst int) *testServer {
	t.Helper()

	d := util.OpenTestDB(t)
	fakeIcecast := util.StartFakeIcecastServer(t)
	sessionManager := session.CreateSessionManager(maxSessions)
	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("create cache: %v", err)
	}
	youtube := music.NewYouTube(t.TempDir(), d, cache)
	testAuth := util.NewTestAuth(t)

	icecastClient := icecast.CreateIcecastClient(
		sessionManager, fakeIcecast.Host, fakeIcecast.Port, "password", "http://icecast-test", d)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if realAudio {
		go icecastClient.StreamSessions(ctx)
	} else {
		go ackSessionsReadyWithoutStreaming(ctx, sessionManager)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	ln.Close()

	httpServer, err := serverconnect.CreateServer("127.0.0.1", port, sessionManager, youtube, icecastClient, testAuth.Auth, d, cache, rateLimitRPS, rateLimitBurst)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := protoconnect.NewRadioServiceClient(http.DefaultClient, baseURL)

	waitFor(t, 2*time.Second, func() bool {
		_, err := client.Ping(context.Background(), connect.NewRequest(&proto.PingRequest{}))
		return err == nil
	})

	return &testServer{client: client, db: d, sessionManager: sessionManager, icecast: fakeIcecast, auth: testAuth}
}

// ackSessionsReadyWithoutStreaming immediately acks CreateSession's readiness signal
// without starting any real playback, standing in for icecast.StreamSessions in tests that
// only care about the RPC/queue surface.
func ackSessionsReadyWithoutStreaming(ctx context.Context, sm *session.SessionManager) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-sm.Events:
			if !ok {
				return
			}
			if event.Type == session.SessionCreated {
				event.Ready <- nil
			}
		}
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

// seedTrack creates a track backed by a real, decodable audio file and enqueues it
// directly on the session's queue, bypassing the AddTrack RPC (which shells out to yt-dlp
// against the real network — not suitable for an automated test).
func seedTrack(t *testing.T, ts *testServer, sessionID session.SessionID, sourceID string) *db.Track {
	t.Helper()
	audioPath := util.GenerateSilentOpus(t, 0.5)
	track, err := ts.db.CreateTrack(context.Background(), "test", sourceID, "Title "+sourceID, "Artist", audioPath, 1, "https://example.com/"+sourceID+".jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	queue, err := ts.sessionManager.GetQueue(sessionID)
	if err != nil {
		t.Fatalf("get queue: %v", err)
	}
	if err := queue.Enqueue(track); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	return track
}

func TestCreateSessionStreamsRealAudio(t *testing.T) {
	ts := setupTestServer(t, true, 1000, 1000, 1000)
	ctx := context.Background()

	resp, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{
		SessionId: "test-session",
		Archive:   true,
	}))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if resp.Msg.StreamUrl != "http://icecast-test/stream/test-session" {
		t.Fatalf("unexpected stream url: %s", resp.Msg.StreamUrl)
	}

	waitFor(t, 2*time.Second, func() bool {
		return ts.icecast.Mountpoint() == "/stream/test-session"
	})

	archives, err := ts.client.ListSessionArchives(ctx, connect.NewRequest(&proto.ListSessionArchivesRequest{}))
	if err != nil || len(archives.Msg.Archives) != 1 {
		t.Fatalf("expected 1 archive, got %v, err %v", archives, err)
	}
	archiveID := archives.Msg.Archives[0].Id

	seedTrack(t, ts, "test-session", "track1")

	// Real ffmpeg decodes the seeded track and real ffmpeg re-encodes it to Ogg/Opus,
	// which the fake Icecast server receives over a real TCP connection.
	waitFor(t, 3*time.Second, ts.icecast.ReceivedOggStream)

	// The icecast playback loop records the play the moment it actually starts streaming.
	waitFor(t, 3*time.Second, func() bool {
		got, err := ts.client.GetSessionArchive(ctx, connect.NewRequest(&proto.GetSessionArchiveRequest{Id: archiveID}))
		return err == nil && len(got.Msg.Tracks) == 1
	})
}

func TestQueueOperations(t *testing.T) {
	ts := setupTestServer(t, false, 1000, 1000, 1000)
	ctx := context.Background()

	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "queue-session"})); err != nil {
		t.Fatalf("create session: %v", err)
	}

	seedTrack(t, ts, "queue-session", "track1")
	seedTrack(t, ts, "queue-session", "track2")

	listResp, err := ts.client.ListQueue(ctx, connect.NewRequest(&proto.ListQueueRequest{SessionId: "queue-session"}))
	if err != nil || len(listResp.Msg.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %v, err %v", listResp, err)
	}

	if _, err := ts.client.RemoveTrack(ctx, connect.NewRequest(&proto.RemoveTrackRequest{SessionId: "queue-session", Index: 0})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument removing index 0, got %v", err)
	}

	if _, err := ts.client.RemoveTrack(ctx, connect.NewRequest(&proto.RemoveTrackRequest{SessionId: "queue-session", Index: 1})); err != nil {
		t.Fatalf("remove track: %v", err)
	}

	listResp, err = ts.client.ListQueue(ctx, connect.NewRequest(&proto.ListQueueRequest{SessionId: "queue-session"}))
	if err != nil || len(listResp.Msg.Tracks) != 1 {
		t.Fatalf("expected 1 track after remove, got %v, err %v", listResp, err)
	}

	if _, err := ts.client.SkipTrack(ctx, connect.NewRequest(&proto.SkipTrackRequest{SessionId: "queue-session"})); err != nil {
		t.Fatalf("skip track: %v", err)
	}
}

func TestSessionNotFound(t *testing.T) {
	ts := setupTestServer(t, false, 1000, 1000, 1000)
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
	}{
		{"GetSession", func() error {
			_, err := ts.client.GetSession(ctx, connect.NewRequest(&proto.GetSessionRequest{SessionId: "nonexistent"}))
			return err
		}},
		{"ListQueue", func() error {
			_, err := ts.client.ListQueue(ctx, connect.NewRequest(&proto.ListQueueRequest{SessionId: "nonexistent"}))
			return err
		}},
		{"SkipTrack", func() error {
			_, err := ts.client.SkipTrack(ctx, connect.NewRequest(&proto.SkipTrackRequest{SessionId: "nonexistent"}))
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.call(); connect.CodeOf(err) != connect.CodeNotFound {
				t.Fatalf("expected NotFound, got %v", err)
			}
		})
	}
}

func TestAuthFlow(t *testing.T) {
	ts := setupTestServer(t, false, 1000, 1000, 1000)
	ctx := context.Background()

	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "auth-session"})); err != nil {
		t.Fatalf("create session: %v", err)
	}

	unauthedReq := connect.NewRequest(&proto.DeleteSessionAuthRequest{SessionId: "auth-session"})
	if _, err := ts.client.DeleteSessionAuth(ctx, unauthedReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected Unauthenticated without a token, got %v", err)
	}

	nonceResp, err := ts.client.RequestNonce(ctx, connect.NewRequest(&proto.RequestNonceRequest{}))
	if err != nil {
		t.Fatalf("request nonce: %v", err)
	}
	passKey := ts.auth.SignNonce(nonceResp.Msg.Nonce)

	tokenResp, err := ts.client.RespondNonce(ctx, connect.NewRequest(&proto.RespondNonceRequest{PassKey: passKey}))
	if err != nil {
		t.Fatalf("respond nonce: %v", err)
	}

	authedReq := connect.NewRequest(&proto.DeleteSessionAuthRequest{SessionId: "auth-session"})
	authedReq.Header().Set("Authorization", "Bearer "+tokenResp.Msg.AuthToken)
	if _, err := ts.client.DeleteSessionAuth(ctx, authedReq); err != nil {
		t.Fatalf("delete session with valid token: %v", err)
	}

	if _, err := ts.client.GetSession(ctx, connect.NewRequest(&proto.GetSessionRequest{SessionId: "auth-session"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected session to be gone, got %v", err)
	}
}

func TestSessionCap(t *testing.T) {
	ts := setupTestServer(t, false, 2, 1000, 1000)
	ctx := context.Background()

	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "cap-session-1"})); err != nil {
		t.Fatalf("create session 1: %v", err)
	}
	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "cap-session-2"})); err != nil {
		t.Fatalf("create session 2: %v", err)
	}
	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "cap-session-3"})); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("expected ResourceExhausted once session cap is hit, got %v", err)
	}
}

func TestRateLimiting(t *testing.T) {
	ts := setupTestServer(t, false, 1000, 1, 2)
	ctx := context.Background()

	newReq := func(ip string) *connect.Request[proto.CreateSessionRequest] {
		req := connect.NewRequest(&proto.CreateSessionRequest{SessionId: fmt.Sprintf("rl-session-%s-%d", ip, time.Now().UnixNano())})
		req.Header().Set("X-Forwarded-For", ip)
		return req
	}

	// Burst of 2 should succeed immediately for a single client IP.
	if _, err := ts.client.CreateSession(ctx, newReq("1.2.3.4")); err != nil {
		t.Fatalf("create session 1: %v", err)
	}
	if _, err := ts.client.CreateSession(ctx, newReq("1.2.3.4")); err != nil {
		t.Fatalf("create session 2: %v", err)
	}
	// Third immediate request from the same IP exceeds the burst.
	if _, err := ts.client.CreateSession(ctx, newReq("1.2.3.4")); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("expected ResourceExhausted once rate limit is hit, got %v", err)
	}

	// A different client IP has its own independent budget.
	if _, err := ts.client.CreateSession(ctx, newReq("5.6.7.8")); err != nil {
		t.Fatalf("expected a different IP to have its own budget, got %v", err)
	}
}

func TestSessionArchiveLifecycle(t *testing.T) {
	ts := setupTestServer(t, false, 1000, 1000, 1000)
	ctx := context.Background()

	if _, err := ts.client.CreateSession(ctx, connect.NewRequest(&proto.CreateSessionRequest{SessionId: "archive-session", Archive: true})); err != nil {
		t.Fatalf("create session: %v", err)
	}

	listResp, err := ts.client.ListSessionArchives(ctx, connect.NewRequest(&proto.ListSessionArchivesRequest{}))
	if err != nil || len(listResp.Msg.Archives) != 1 {
		t.Fatalf("expected 1 archive, got %v, err %v", listResp, err)
	}
	archiveID := listResp.Msg.Archives[0].Id

	if _, err := ts.client.GetSessionArchive(ctx, connect.NewRequest(&proto.GetSessionArchiveRequest{Id: archiveID})); err != nil {
		t.Fatalf("get archive: %v", err)
	}

	unauthedReq := connect.NewRequest(&proto.DeleteSessionArchiveRequest{Id: archiveID})
	if _, err := ts.client.DeleteSessionArchive(ctx, unauthedReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected Unauthenticated without a token, got %v", err)
	}

	nonceResp, err := ts.client.RequestNonce(ctx, connect.NewRequest(&proto.RequestNonceRequest{}))
	if err != nil {
		t.Fatalf("request nonce: %v", err)
	}
	tokenResp, err := ts.client.RespondNonce(ctx, connect.NewRequest(&proto.RespondNonceRequest{PassKey: ts.auth.SignNonce(nonceResp.Msg.Nonce)}))
	if err != nil {
		t.Fatalf("respond nonce: %v", err)
	}

	authedReq := connect.NewRequest(&proto.DeleteSessionArchiveRequest{Id: archiveID})
	authedReq.Header().Set("Authorization", "Bearer "+tokenResp.Msg.AuthToken)
	if _, err := ts.client.DeleteSessionArchive(ctx, authedReq); err != nil {
		t.Fatalf("delete archive with valid token: %v", err)
	}

	if _, err := ts.client.GetSessionArchive(ctx, connect.NewRequest(&proto.GetSessionArchiveRequest{Id: archiveID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected archive to be gone, got %v", err)
	}
}
