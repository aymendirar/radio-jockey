package util

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"server/src/connect/auth"
	"server/src/db"
	"strings"
	"sync"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
)

func OpenTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// GenerateSilentOpus renders a short silent opus file via ffmpeg, for tests that need a
// real, decodable audio file without depending on a network download.
func GenerateSilentOpus(t *testing.T, seconds float64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "silence.opus")
	cmd := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", fmt.Sprintf("anullsrc=r=48000:cl=stereo:d=%f", seconds),
		"-c:a", "libopus",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate test audio: %v: %s", err, out)
	}
	return path
}

// TestAuth wraps a fresh, in-memory PASETO keypair so integration tests can both run the
// server's Auth verification and sign nonces themselves, standing in for the external
// signer (the Discord bot / signnonce CLI in production).
type TestAuth struct {
	*auth.Auth
	privateKey paseto.V4AsymmetricSecretKey
}

func NewTestAuth(t *testing.T) *TestAuth {
	t.Helper()
	sk := paseto.NewV4AsymmetricSecretKey()
	privatePASERK := "k4.secret." + base64.RawURLEncoding.EncodeToString(sk.ExportBytes())
	publicPASERK := "k4.public." + base64.RawURLEncoding.EncodeToString(sk.Public().ExportBytes())

	a, err := auth.NewAuth(privatePASERK, publicPASERK, 2*time.Minute)
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}
	return &TestAuth{Auth: a, privateKey: sk}
}

// SignNonce signs a server-issued nonce the same way the Discord bot / signnonce CLI would,
// producing a passkey that RespondNonce will accept.
func (a *TestAuth) SignNonce(nonce string) string {
	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(2 * time.Minute))
	token.SetString("nonce", nonce)
	return token.V4Sign(a.privateKey, nil)
}

// FakeIcecastServer stands in for a real Icecast server: it accepts the same raw HTTP PUT
// source-client handshake IcecastClient speaks, then records the mountpoint and streamed
// bytes. This lets tests exercise the real icecast package (real TCP dial, real ffmpeg
// encode) without mocking any Go code.
type FakeIcecastServer struct {
	Host string
	Port string

	mu         sync.Mutex
	mountpoint string
	received   []byte
}

const fakeIcecastMaxBuffered = 1 << 16

func StartFakeIcecastServer(t *testing.T) *FakeIcecastServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	f := &FakeIcecastServer{Host: host, Port: port}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go f.handle(conn)
		}
	}()

	return f
}

func (f *FakeIcecastServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	fields := strings.Fields(requestLine)
	if len(fields) < 2 {
		return
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}

	if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		return
	}

	f.mu.Lock()
	f.mountpoint = fields[1]
	f.mu.Unlock()

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			f.mu.Lock()
			if len(f.received) < fakeIcecastMaxBuffered {
				f.received = append(f.received, buf[:n]...)
			}
			f.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (f *FakeIcecastServer) Mountpoint() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mountpoint
}

// ReceivedOggStream reports whether a valid Ogg container ("OggS" magic) has appeared in
// the streamed bytes, i.e. the real ffmpeg encoder pipeline is actually producing output.
func (f *FakeIcecastServer) ReceivedOggStream() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return bytes.Contains(f.received, []byte("OggS"))
}
