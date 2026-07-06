package music_test

import (
	"server/src/music"
	"testing"
)

func TestIsYouTubeURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"bare youtube.com", "https://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"www.youtube.com", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"mobile m.youtube.com", "https://m.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"music.youtube.com", "https://music.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"shortened youtu.be", "https://youtu.be/dQw4w9WgXcQ", true},
		{"shortened youtu.be with query", "https://youtu.be/dQw4w9WgXcQ?t=30", true},
		{"privacy-enhanced embed domain", "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ", true},
		{"bare youtube-nocookie.com", "https://youtube-nocookie.com/embed/dQw4w9WgXcQ", true},
		{"shorts path on youtube.com", "https://youtube.com/shorts/dQw4w9WgXcQ", true},
		{"uppercase host", "https://YOUTUBE.COM/watch?v=dQw4w9WgXcQ", true},
		{"explicit port", "https://youtube.com:443/watch?v=dQw4w9WgXcQ", true},

		{"unrelated domain", "https://example.com/video", false},
		{"lookalike suffix domain", "https://youtube.com.evil.com/watch?v=dQw4w9WgXcQ", false},
		{"lookalike path", "https://evil.com/youtube.com", false},
		{"internal app url", "https://localhost:3000/stations/lovefield", false},
		{"empty string", "", false},
		{"not a url", "not a url", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := music.IsYouTubeURL(c.url); got != c.want {
				t.Errorf("IsYouTubeURL(%q) = %v, want %v", c.url, got, c.want)
			}
		})
	}
}
