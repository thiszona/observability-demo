// Package httpx provides a trace-instrumented HTTP client.
package httpx

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client returns an http.Client whose transport is instrumented by otelhttp, so
// each outgoing request gets a client span and propagates the trace context.
func Client() *http.Client {
	return &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
}
