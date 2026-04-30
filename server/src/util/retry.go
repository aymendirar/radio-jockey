package util

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/avast/retry-go"
)

var Unrecoverable = retry.Unrecoverable

func RetryWithBackoff(ctx context.Context, fn retry.RetryableFunc, onRetry retry.OnRetryFunc) error {
	err := retry.Do(
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
	// retry.Error is []error with no Unwrap(), which breaks errors.Is/As for callers.
	// Return the last underlying error directly to preserve the error chain.
	var retryErrs retry.Error
	if errors.As(err, &retryErrs) && len(retryErrs) > 0 {
		for i := len(retryErrs) - 1; i >= 0; i-- {
			if retryErrs[i] != nil {
				return retryErrs[i]
			}
		}
	}
	return err
}
