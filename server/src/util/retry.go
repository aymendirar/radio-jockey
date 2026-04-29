package util

import (
	"context"
	"log/slog"
	"time"

	"github.com/avast/retry-go"
)

func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return retry.Do(
		fn,
		retry.Context(ctx),
		retry.Attempts(10),
		retry.Delay(time.Second),
		retry.MaxDelay(32*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			slog.Warn("retrying", "attempt", n+1, "err", err)
		}),
	)
}
