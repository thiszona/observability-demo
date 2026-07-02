// Package chaos provides env-driven HTTP fault injection (latency and errors)
// for the demo services. Every knob defaults to off.
package chaos

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the fault-injection knobs, read from the environment.
type Config struct {
	Latency   time.Duration // CHAOS_LATENCY, e.g. "250ms"
	ErrorRate float64       // CHAOS_ERROR_RATE, 0.0..1.0
	Status    int           // CHAOS_ERROR_STATUS, default 500
}

// FromEnv reads the CHAOS_* variables. Missing or invalid values are left at zero.
func FromEnv() Config {
	c := Config{Status: 500}
	if v := os.Getenv("CHAOS_LATENCY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Latency = d
		}
	}
	if v := os.Getenv("CHAOS_ERROR_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.ErrorRate = f
		}
	}
	if v := os.Getenv("CHAOS_ERROR_STATUS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Status = n
		}
	}
	return c
}

// Middleware injects the configured latency and errors. It must run inside
// otelhttp.NewHandler so the active span is in the request context; it tags that
// span with any injected fault.
func (c Config) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())

		if c.Latency > 0 {
			span.SetAttributes(attribute.Int64("chaos.latency_ms", c.Latency.Milliseconds()))
			select {
			case <-time.After(c.Latency):
			case <-r.Context().Done():
				// caller canceled during the injected delay
				span.SetStatus(codes.Error, "canceled during chaos latency")
				http.Error(w, "timeout", http.StatusGatewayTimeout)
				return
			}
		}

		// math/rand is sufficient for a demo
		if c.ErrorRate > 0 && rand.Float64() < c.ErrorRate {
			span.SetAttributes(
				attribute.Bool("chaos.injected_error", true),
				attribute.Float64("chaos.error_rate", c.ErrorRate),
			)
			span.SetStatus(codes.Error, "chaos injected error")
			http.Error(w, "chaos", c.Status)
			return
		}

		next.ServeHTTP(w, r)
	})
}
