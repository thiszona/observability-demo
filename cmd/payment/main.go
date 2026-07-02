// payment sits between checkout and fraud. It forwards to fraud, and is the
// service the compose file points chaos at (CHAOS_ERROR_RATE) by default.
package main

import (
	"observability-demo/internal/httpx"
	"observability-demo/internal/service"
)

func main() {
	client := httpx.Client()
	service.Run("payment", service.Forward(client, service.EnvOr("FRAUD_URL", "http://fraud:8080")))
}
