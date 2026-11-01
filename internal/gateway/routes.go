package gateway

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(
	app *fiber.App,
	proxy *Proxy,
) {
	app.All("/gateway/users/*", proxy.Handler("users"))
	app.All("/gateway/products/*", proxy.Handler("products"))
	app.All("/gateway/orders/*", proxy.Handler("orders"))
}
