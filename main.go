package main

import (
	"log"

	router "github.com/axuman/go-server/api"
	G "github.com/axuman/go-server/globals"
	svr "github.com/axuman/go-server/svr"

	"github.com/gofiber/fiber/v2"
)

var err error

func main() {

	// db
	G.DmailDB, err = svr.InitDB("./dmail.db")
	if err != nil {
		log.Fatal(err)
	}
	defer G.DmailDB.Close()


	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Printf("Unhandled error: %v - Path: %s", err, c.Path())
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	// app.Use(logger.New())
	// app.Use(recover.New())

	router.BuildRoutes(app)

	if err := app.Listen(":3001"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
