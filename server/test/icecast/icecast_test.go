package icecast_test

import (
	"server/src/icecast"
	"server/src/session"
	"testing"
)

func TestStreamURL(t *testing.T) {
	client := icecast.CreateIcecastClient(session.CreateSessionManager(1000), "icecast", "9999", "pass", "http://icecast:9999", nil)

	cases := map[session.SessionID]string{
		"my-session": "http://icecast:9999/stream/my-session",
		"other":      "http://icecast:9999/stream/other",
	}
	for sessionID, want := range cases {
		if got := client.StreamURL(sessionID); got != want {
			t.Fatalf("StreamURL(%q) = %q, want %q", sessionID, got, want)
		}
	}
}
