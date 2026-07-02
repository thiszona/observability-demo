// injector is a load generator: it fires one request at the gateway every
// INJECT_INTERVAL. Each request starts its own trace.
package main

import (
	"context"
	"net/http"
	"time"

	"observability-demo/internal/httpx"
	"observability-demo/internal/logx"
	"observability-demo/internal/otelinit"
	"observability-demo/internal/service"
)

func main() {
	ctx := context.Background()
	log := logx.New("injector")

	shutdown, err := otelinit.Init(ctx, "injector")
	if err != nil {
		log.Error("otel init failed", "err", err)
		return
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
			log.ErrorContext(ctx, "request failed", "err", err)
			continue
		}
		_ = resp.Body.Close()
		log.InfoContext(ctx, "request done", "status", resp.StatusCode)
	}
}
