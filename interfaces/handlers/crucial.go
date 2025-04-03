package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func Limiter() fiber.Handler {

	limits := limiter.Config{
		// Next: func(c *fiber.Ctx) bool {
		// 	return c.IP() == "127.0.0.1" //  || c.IP() == "0.0.0.0"
		// },
		Max:        20,
		Expiration: 20 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Get("x-forwarded-for")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendFile("views/toofast.html")
		},
		// Storage: myCustomStorage{},
	}

	return limiter.New(limits)
}
