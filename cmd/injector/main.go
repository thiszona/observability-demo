// injector is a load generator: it fires one request at the gateway every
// INJECT_INTERVAL. Each request starts its own trace.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"observability-demo/internal/httpx"
	"observability-demo/internal/otelinit"
	"observability-demo/internal/service"
)

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "injector")

	shutdown, err := otelinit.Init(ctx, "injector")
	if err != nil {
		log.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	client := httpx.Client()
	gateway := service.EnvOr("GATEWAY_URL", "http://gateway:8080")
	interval, err := time.ParseDuration(service.EnvOr("INJECT_INTERVAL", "500ms"))
	if err != nil {
		interval = 500 * time.Millisecond
	}

	log.Info("injecting traffic", "target", gateway, "interval", interval.String())
	for range time.Tick(interval) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, gateway, nil)
		resp, err := client.Do(req)
		if err != nil {
			log.Error("request failed", "err", err)
			continue
		}
		_ = resp.Body.Close()
		log.Info("request done", "status", resp.StatusCode)
	}
}
