// gateway is the entry point. It forwards each request to checkout.
package main

import (
	"observability-demo/internal/httpx"
	"observability-demo/internal/service"
)

func main() {
	client := httpx.Client()
	service.Run("gateway", service.Forward(client, service.EnvOr("CHECKOUT_URL", "http://checkout:8080")))
}
