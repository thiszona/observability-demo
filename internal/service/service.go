// Package service boots an HTTP service with OpenTelemetry, chaos injection, and
// graceful shutdown, so each command only needs to supply a handler.
package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"observability-demo/internal/chaos"
	"observability-demo/internal/logx"
	"observability-demo/internal/otelinit"
)

// Handler handles a request. Use log.InfoContext/ErrorContext with r.Context() so
// the log line is correlated with the request's trace.
type Handler func(w http.ResponseWriter, r *http.Request, log *slog.Logger)

// Run starts the named service: OpenTelemetry, the chaos middleware, an
// otelhttp-wrapped handler on PORT (default 8080), and graceful shutdown.
func Run(name string, handle Handler) {
	ctx := context.Background()
	logger := logx.New(name)

	shutdown, err := otelinit.Init(ctx, name)
	if err != nil {
		logger.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, logger)
	})

	// otelhttp (outer) starts the span, chaos injects and tags it, mux handles.
	handler := otelhttp.NewHandler(chaos.FromEnv().Middleware(mux), name)

	srv := &http.Server{Addr: ":" + EnvOr("PORT", "8080"), Handler: handler}
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")
	_ = srv.Shutdown(ctx)
}

// Forward returns a handler that calls the next service and mirrors its result.
// On a 5xx it records the error on the current span so the failure propagates up
// the trace.
func Forward(client *http.Client, next string) Handler {
	return func(w http.ResponseWriter, r *http.Request, log *slog.Logger) {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, next, nil)
		resp, err := client.Do(req)
		if err != nil {
			span := trace.SpanFromContext(r.Context())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(r.Context(), "downstream call failed", "url", next, "err", err)
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			span := trace.SpanFromContext(r.Context())
			e := fmt.Errorf("downstream %s returned %d", next, resp.StatusCode)
			span.RecordError(e)
			span.SetStatus(codes.Error, e.Error())
			log.ErrorContext(r.Context(), "downstream error", "url", next, "status", resp.StatusCode)
			http.Error(w, "downstream failed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "ok")
	}
}

// EnvOr returns the value of environment variable k, or def if k is unset.
func EnvOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
