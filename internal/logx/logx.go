// Package logx provides structured logging that goes to both stdout and OTLP,
// correlated with the active trace. Use the *Context methods (InfoContext,
// ErrorContext) so the trace id reaches both sinks.
package logx

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
)

// New returns a logger tagged with the service name. Records are written as JSON
// to stdout (with the calling context's trace_id) and exported over OTLP through
// the OpenTelemetry slog bridge.
func New(service string) *slog.Logger {
	h := tee{[]slog.Handler{
		traceHandler{slog.NewJSONHandler(os.Stdout, nil)},
		otelslog.NewHandler(service),
	}}
	return slog.New(h).With("service", service)
}

// traceHandler adds the context's trace_id to each stdout record. The OTLP bridge
// already carries trace context on its own.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(as []slog.Attr) slog.Handler {
	return traceHandler{h.Handler.WithAttrs(as)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{h.Handler.WithGroup(name)}
}

// tee fans each record out to several handlers.
type tee struct{ hs []slog.Handler }

func (t tee) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range t.hs {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (t tee) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range t.hs {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (t tee) WithAttrs(as []slog.Attr) slog.Handler {
	ns := make([]slog.Handler, len(t.hs))
	for i, h := range t.hs {
		ns[i] = h.WithAttrs(as)
	}
	return tee{ns}
}

func (t tee) WithGroup(name string) slog.Handler {
	ns := make([]slog.Handler, len(t.hs))
	for i, h := range t.hs {
		ns[i] = h.WithGroup(name)
	}
	return tee{ns}
}
