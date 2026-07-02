// Package logx provides structured logging correlated with the active trace.
package logx

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// New returns a JSON logger tagged with the service name.
func New(service string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", service)
}

// With returns a logger that adds the context's trace_id to every line, so logs
// can be correlated with their trace.
func With(ctx context.Context, l *slog.Logger) *slog.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return l
	}
	return l.With("trace_id", sc.TraceID().String())
}
