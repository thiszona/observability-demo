// fraud is the leaf service, standing in for a risk check. On its own it returns
// ok; the chaos knobs (CHAOS_LATENCY, CHAOS_ERROR_RATE) are what make it misbehave.
package main

import (
	"io"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"observability-demo/internal/service"
)

func main() {
	tracer := otel.Tracer("fraud")

	service.Run("fraud", func(w http.ResponseWriter, r *http.Request, log *slog.Logger) {
		// The auto HTTP span covers the request; add a child span for the risk check.
		_, span := tracer.Start(r.Context(), "risk-check")
		defer span.End()
		span.SetAttributes(attribute.String("user.tier", "standard"))

		log.InfoContext(r.Context(), "risk check ok")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
}
