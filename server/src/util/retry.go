package util

import (
	"context"
	"log/slog"
	"time"

	"github.com/avast/retry-go"
)

var Unrecoverable = retry.Unrecoverable

func RetryWithBackoff(ctx context.Context, fn retry.RetryableFunc, onRetry retry.OnRetryFunc) error {
	return retry.Do(
		fn,
		retry.Context(ctx),
		retry.Attempts(10),
		retry.Delay(time.Second),
		retry.MaxDelay(32*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			onRetry(n, err)
			slog.Warn("retrying...", "attempt", n+1, "err", err)
		}),
	)
}
