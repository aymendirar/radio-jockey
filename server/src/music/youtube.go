// TODO
package music

import (
	"github.com/dhowden/tag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type YouTube struct {
	path string
}

func CreateYouTubeClient(path string) (*YouTube, error) {
	return &YouTube{path: path}, nil
}

func (y *YouTube) DownloadTrackFromURL(url string) {
	slog.Info("starting youtube download", "url", url, "path", y.path)
	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format",
		"opus",
		"--no-playlist",
		"--embed-metadata",
		"--print", "after_move:filepath",
		"-o",
		y.path+"/youtube/%(id)s.opus",
		url)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		slog.Error("error running youtube download command", "err", err, "stderr", stderr.String())
		return
	}

	filePath := strings.TrimSpace(string(out))
	f, err := os.Open(filePath)
	if err != nil {
		slog.Error("error opening downloaded file", "err", err, "path", filePath)
		return
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		slog.Error("error extracting tags", "err", err, "path", filePath)
		return
	}
	slog.Info("download complete", "url", url, "title", m.Title(), "path", filePath)
}
