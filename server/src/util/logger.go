package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type handler struct {
	w io.Writer
}

func (h *handler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	attrs := ""
	if r.NumAttrs() > 0 {
		attrs = " {"
		i := 0
		r.Attrs(func(a slog.Attr) bool {
			attrs += fmt.Sprintf(" %s: %v", a.Key, a.Value)
			if i < r.NumAttrs()-1 {
				attrs += ","
			}
			i++
			return true
		})
		attrs += " }"
	}

	fmt.Fprintf(h.w, "[server] %s: %s%s\n", r.Level, r.Message, attrs)
	return nil
}

func (h *handler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *handler) WithGroup(_ string) slog.Handler      { return h }

func NewLogger() *slog.Logger {
	return slog.New(&handler{w: os.Stdout})
}
