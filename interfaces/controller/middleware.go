package controller

import (
	"context"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/gofiber/fiber/v2"
)

// Middleware to verify Firebase JWT Token
func FireAuthMiddleware(firebaseAuth *auth.Client) fiber.Handler {

	return func(c *fiber.Ctx) error {

		// Ignore authentication for / and /metrics paths
		// if c.Path() == "/" || c.Path() == "/metrics" || c.Path() == "/health" || c.Path() == "/metricsgraph" {
		// 	return c.Next()
		// }

		token := c.Get("Authorization")

		// Check if the token starts with "Bearer "
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:] // Extract token
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
		}

		decodedToken, err := firebaseAuth.VerifyIDToken(context.Background(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		c.Locals("user", decodedToken)
		return c.Next()
	}
}

// func FireAuthorize(IDToken string, CurrentURL string) (*auth.Token, error){
// 	// create your own authentication here

// 	if IDToken == "" {
// 		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
// 	}

// 	decodedToken, err := firebaseAuth.VerifyIDToken(context.Background(), token)
// 	if err != nil {
// 		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
// 	}

// 	// this returns the firebase id token
// 	return token, nil
// }
