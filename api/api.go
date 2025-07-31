package api

import (
	user "github.com/axuman/go-server/api/dmail"
	mall "github.com/axuman/go-server/api/dmail/mall"
	"github.com/gofiber/fiber/v2"
)

func BuildRoutes(router fiber.Router) {
	dmail_router := router.Group("/dmail")
	user.BuildRoutes(dmail_router)
	mall.BuildRoutes(dmail_router)

	router.Get("/health", func(c *fiber.Ctx) error {
		c.SendString("OK")
		return nil
	})
}
