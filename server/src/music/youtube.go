package music

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
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
	cache              *Cache
}

const (
	SOURCE = "youtube"
)

var (
	ErrInvalidURL            = errors.New("invalid url")
	ErrVideoUnavailable      = errors.New("video unavailable")
	ErrUnexpectedYtdlpOutput = errors.New("unexpected yt-dlp output")
)

func NewYouTube(musicDirectoryPath string, db *db.DB, cache *Cache) *YouTube {
	return &YouTube{
		musicDirectoryPath: musicDirectoryPath,
		db:                 db,
		cache:              cache}
}

type ytdlpResult struct {
	sourceId string
	duration int64
	filePath string
}

func IsYouTubeURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be",
		"youtube-nocookie.com", "www.youtube-nocookie.com":
		return true
	default:
		return false
	}
}

func (y *YouTube) DownloadTrackFromURL(ctx context.Context, url string) (*db.Track, error) {
	slog.Info("starting youtube download", "url", url, "path", y.musicDirectoryPath)

	if !IsYouTubeURL(url) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidURL, url)
	}

	var sourceId string
	if err := y.withRetry(ctx, "failed to resolve youtube video id", url, func() error {
		var err error
		sourceId, err = y.peekSourceID(url)
		return err
	}); err != nil {
		return nil, fmt.Errorf("resolving video id failed for %s: %w", url, err)
	}

	if track, ok := y.cache.Get(sourceId); ok {
		if _, err := os.Stat(track.FilePath); err == nil {
			slog.Info("cache hit, skipping download", "url", url, "source_id", sourceId, "title", track.Title, "artist", track.Artist)
			return track, nil
		}
	}

	var result ytdlpResult
	if err := y.withRetry(ctx, "failed to download youtube track", url, func() error {
		var err error
		result, err = y.ytdlp(url)
		return err
	}); err != nil {
		return nil, fmt.Errorf("yt-dlp failed for %s: %w", url, err)
	}

	metadata, err := y.extractTagMetadata(result.filePath)
	if err != nil {
		return nil, err
	}

	slog.Info("download complete", "url", url, "title", metadata.Title(), "path", result.filePath)

	albumArtUrl := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", result.sourceId)

	if track, err := y.db.GetTrack(ctx, result.sourceId); err == nil {
		slog.Info("found track in database, not creating record...", "title", track.Title, "artist", track.Artist)
		if err := y.db.UpdateTrackAlbumArtUrl(ctx, track.Id, albumArtUrl); err != nil {
			return nil, err
		}
		track.AlbumArtUrl = albumArtUrl
		return track, nil
	}

	track, err := y.db.CreateTrack(
		ctx,
		SOURCE,
		result.sourceId,
		metadata.Title(),
		metadata.Artist(),
		result.filePath,
		result.duration,
		albumArtUrl)

	slog.Info("created new track record", "title", track.Title, "artist", track.Artist)

	return track, err
}

// classifyYtdlpError maps yt-dlp's stderr output to a sentinel error so callers can
// distinguish a bad/unsupported URL or an unavailable video from a transient failure.
func classifyYtdlpError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := string(exitErr.Stderr)
		if strings.Contains(stderr, "Unsupported URL") || strings.Contains(stderr, "is not a valid URL") {
			return fmt.Errorf("%w: %s", ErrInvalidURL, stderr)
		}
		if strings.Contains(stderr, "Video unavailable") {
			return fmt.Errorf("%w: %s", ErrVideoUnavailable, stderr)
		}
		return fmt.Errorf("%w: %s", err, stderr)
	}
	return err
}

// withRetry wraps op with the standard retry-with-backoff policy, treating a classified
// ErrInvalidURL/ErrVideoUnavailable as unrecoverable rather than retrying it.
func (y *YouTube) withRetry(ctx context.Context, warnMsg, url string, op func() error) error {
	return util.RetryWithBackoff(
		ctx,
		func() error {
			err := op()
			if errors.Is(err, ErrInvalidURL) || errors.Is(err, ErrVideoUnavailable) {
				return util.Unrecoverable(err)
			}
			return err
		},
		func(n uint, err error) {
			slog.Warn(warnMsg, "url", url)
		})
}

func (y *YouTube) ytdlp(url string) (ytdlpResult, error) {
	// will exit early if file already exists in music directory
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

	out, err := cmd.Output()
	if err != nil {
		return ytdlpResult{}, classifyYtdlpError(err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		return ytdlpResult{}, fmt.Errorf("%w: %q", ErrUnexpectedYtdlpOutput, string(out))
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

// peekSourceID resolves a URL's video id via yt-dlp without downloading or converting any
// audio, so a cache hit can be checked before paying for a full download.
func (y *YouTube) peekSourceID(url string) (string, error) {
	cmd := exec.Command(
		"yt-dlp",
		"--skip-download",
		"--no-playlist",
		"--print", "id",
		url,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", classifyYtdlpError(err)
	}

	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf("%w: empty id from yt-dlp", ErrUnexpectedYtdlpOutput)
	}
	return id, nil
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
