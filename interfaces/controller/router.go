package controller

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/AntoniadisCorp/deploy4scrap/domain"
	"github.com/AntoniadisCorp/deploy4scrap/interfaces/handlers"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

func Welcome(c *fiber.Ctx) error {
	return c.SendString("Welcome to the Deploy4Scrap API!")
}

func SetupRoutes(app *fiber.App, fireAuthHandler *handlers.AuthHandler, FirebaseApp *firebase.App) {

	ctx := context.Background()

	firebaseAuth, err := FirebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Firebase Auth initialization error: %v", err)
	}

	midlwrFireAuth := FireAuthMiddleware(firebaseAuth)

	// * Serve static images from a specific directory
	// * Static Files Handler

	// Access file "image.png" under `static/` directory via URL: `http://<server>/assets/image.png`.
	// Without `PathPrefix`, you have to access it via URL:
	// `http://<server>/assets/static/image.png`. Or extend your config for customization

	app.Use("/assets", filesystem.New(filesystem.Config{
		Root:         http.Dir("./assets"),
		Browse:       false,
		Index:        "views/index.html",
		NotFoundFile: "views/404.html",
		MaxAge:       3600,
	}))

	// Apply rate limiting to all routes
	// app.Use(middleware.RateLimiter(cfg.RateLimit, cfg.RateLimitDuration))

	app.Use(etag.New())

	// Or extend your config for customization
	app.Use(favicon.New(favicon.Config{
		File: "./favicon.ico",
		URL:  "/favicon.ico",
	}))

	// Initialize Prometheus
	prometheus := fiberprometheus.New("Deploy4Scrap")
	prometheus.RegisterAt(app, "/metrics")
	// prometheus.SetSkipPaths([]string{"/ping"}) // Optional: Remove some paths from metrics

	// Initialize the Prometheus middleware
	app.Use(prometheus.Middleware)

	// Provide a minimal config
	app.Use(healthcheck.New())

	// Initialize default config
	app.Use(handlers.Limiter())

	// handle Routers Status NotFound Failure
	app.Use(handleStatusNotFoundFailure())

	// oauth Authorization Failure
	// app.Get("/failure", MakeServerCallback(authHandler.HandleFailure))
	// app.Get("/success", MakeServerCallback(authHandler.HandleSuccess))

	// Routes that don't require authentication
	app.Get("/", Welcome)

	// Start Metrics server
	app.Get("/metricsgraph", monitor.New(monitor.Config{Title: "Deploy4Scrap Metrics Page"}))

	// Create a group for authenticated routes
	authedApp := app.Group("/api", midlwrFireAuth)

	authedApp.Post("/deploy", MakeFiberCallback(fireAuthHandler.DeployMachine))
	authedApp.Put("/machine/:id/start", MakeFiberCallback(fireAuthHandler.StartMachine))
	authedApp.Put("/machine/:id/stop", MakeFiberCallback(fireAuthHandler.StopMachine))
	authedApp.Delete("/machine/:id", MakeFiberCallback(fireAuthHandler.DeleteMachine))
	// authedApp.Post("/execute-task/:machine_id", executeTask)

}

func StatusNotFound(c *fiber.Ctx) (*domain.HTTPResponse, error) {

	response := domain.HTTPResponse{
		// Headers: headers,
		Code:    fiber.StatusNotFound,
		Status:  "forbidden",
		Message: "Not Found",

		Data: fiber.Map{
			"Error":   "Not Found",
			"Message": "The requested resource could not be found",
			"Code":    "[8c8f6eb1890a9249-FRA]",
		},
		// TraceID: traceID,
		Errors: domain.APIError{Code: "[8c8f6eb1890a9249-FRA]", Message: "The requested resource could not be found"},
		View:   "403",
	}

	return &response, nil
}

func SetupNonAccessibleRoutes(app *fiber.App) {
	app.Get("/"+strconv.Itoa(fiber.StatusNotFound), MakeServerCallback(StatusNotFound))
}

func handleStatusNotFoundFailure() func(c *fiber.Ctx) error {

	return func(c *fiber.Ctx) error {

		if c.Path() == "/" || strings.Contains(c.Path(), ".js") || strings.Contains(c.Path(), ".css") ||
			strings.Contains(c.Path(), ".jpg") || strings.Contains(c.Path(), ".png") ||
			strings.Contains(c.Path(), ".svg") || strings.Contains(c.Path(), ".ico") ||
			strings.Contains(c.Path(), ".gif") || strings.Contains(c.Path(), ".jpeg") ||
			strings.Contains(c.Path(), ".bmp") || strings.Contains(c.Path(), ".webp") ||
			strings.Contains(c.Path(), ".avif") || strings.Contains(c.Path(), ".woff2") {

			c.Type("html")
			return c.Next()
		}

		// reports whether the string s begins with prefix. then return next handler
		if strings.HasPrefix(c.Path(), "/api") || strings.HasPrefix(c.Path(), "/health") || strings.HasPrefix(c.Path(), "/metrics") {
			c.Type("json")
			return c.Next()
		}

		// Render a 404 Not Found response
		httpResponse := domain.HTTPResponse{
			Code:    fiber.StatusNotFound,
			Status:  "Not Found",
			Message: "Page Not Found",
			Data: fiber.Map{
				"Error":   "Not Found",
				"Message": "The requested resource could not be found.",
				"Code":    "[8c8f6eb1890a9249-FRA]",
			},
			// SubDomainHost: string(c.Context().Host()),
			// DomainHost: string(c.Context().Host()),
			Errors: domain.APIError{Code: "[8c8f6eb1890a9249-FRA]", Message: "This is not the web page you are looking for."},
			View:   "404",
		}

		c.Type("html")

		return c.Status(httpResponse.Code).Render(httpResponse.View, httpResponse)
	}

}
