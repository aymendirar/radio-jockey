package music

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"server/src/db"
	"strconv"
	"strings"

	"server/src/util"

	"github.com/dhowden/tag"
)

type YouTube struct {
	musicDirectoryPath string
	db                 *db.DB
}

const (
	SOURCE = "youtube"
)

func NewYouTube(musicDirectoryPath string, db *db.DB) *YouTube {
	return &YouTube{
		musicDirectoryPath: musicDirectoryPath,
		db:                 db}
}

type ytdlpResult struct {
	sourceId string
	duration int64
	filePath string
}

func (y *YouTube) DownloadTrackFromURL(ctx context.Context, url string) (*db.Track, error) {
	slog.Info("starting youtube download", "url", url, "path", y.musicDirectoryPath)

	var result ytdlpResult
	err := util.RetryWithBackoff(
		ctx,
		func() error {
			var err error
			result, err = y.ytdlp(url)
			return err
		},
		func(n uint, err error) {
			slog.Warn("failed to download youtube track", "url", url)
		})
	if err != nil {
		return nil, fmt.Errorf("yt-dlp failed for %s: %w", url, err)
	}

	metadata, err := y.extractTagMetadata(result.filePath)
	if err != nil {
		return nil, err
	}

	slog.Info("download complete", "url", url, "title", metadata.Title(), "path", result.filePath)

	if track, err := y.db.GetTrack(ctx, result.sourceId); err == nil {
		slog.Info("found track in database, not creating record...", "title", track.Title, "artist", track.Artist)
		return track, nil
	}

	track, err := y.db.CreateTrack(
		ctx,
		SOURCE,
		result.sourceId,
		metadata.Title(),
		metadata.Artist(),
		result.filePath,
		result.duration)

	slog.Info("created new track record", "title", track.Title, "artist", track.Artist)

	return track, err
}

func (y *YouTube) ytdlp(url string) (ytdlpResult, error) {
	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "opus",
		"--no-playlist",
		"--embed-metadata",
		"--print", "id",
		"--print", "duration",
		"--print", "after_move:filepath",
		"-o", filepath.Join(y.musicDirectoryPath, "youtube", "%(id)s.opus"),
		url,
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return ytdlpResult{}, fmt.Errorf("%w: %s", err, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		return ytdlpResult{}, fmt.Errorf("unexpected yt-dlp output: %q", string(out))
	}

	d, err := strconv.ParseFloat(strings.TrimSpace(lines[1]), 64)
	if err != nil {
		return ytdlpResult{}, fmt.Errorf("parsing duration %q: %w", lines[1], err)
	}
	duration := int64(d)

	return ytdlpResult{
		sourceId: strings.TrimSpace(lines[0]),
		duration: duration,
		filePath: strings.TrimSpace(lines[2]),
	}, nil
}

func (y *YouTube) extractTagMetadata(filePath string) (tag.Metadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening downloaded file: %w", err)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		slog.Warn("could not extract tags", "err", err, "path", filePath)
		return nil, err
	}

	return m, nil
}
