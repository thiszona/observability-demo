// checkout orchestrates the purchase by calling payment. It also owns the custom
// metrics: checkout duration and failure count.
package main

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"observability-demo/internal/httpx"
	"observability-demo/internal/service"
)

func main() {
	client := httpx.Client()
	payment := service.EnvOr("PAYMENT_URL", "http://payment:8080")

	// A meter provides instruments: a histogram for a distribution (durations),
	// a counter for a tally (failures).
	meter := otel.Meter("checkout")
	duration, _ := meter.Float64Histogram("checkout.duration", metric.WithUnit("ms"))
	requests, _ := meter.Int64Counter("checkout.requests")
	failures, _ := meter.Int64Counter("checkout.failures")

	service.Run("checkout", func(w http.ResponseWriter, r *http.Request, log *slog.Logger) {
		start := time.Now()
		requests.Add(r.Context(), 1)

		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, payment, nil)
		resp, err := client.Do(req)
		ok := err == nil && resp.StatusCode < 500
		if resp != nil {
			_ = resp.Body.Close()
		}

		// Label with the route only, never a high-cardinality value like a user id.
		duration.Record(r.Context(), float64(time.Since(start).Milliseconds()),
			metric.WithAttributes(attribute.String("route", "/checkout")))

		if !ok {
			failures.Add(r.Context(), 1)
			log.ErrorContext(r.Context(), "payment failed", "err", err)
			http.Error(w, "checkout failed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "ok")
	})
}
