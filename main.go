package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AntoniadisCorp/deploy4scrap/handlers"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/goccy/go-json"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Define a struct to hold the query parameters
type DeployQuery struct {
	Clone    bool   `query:"clone"`
	MasterID string `query:"master_id"`
}

var firebaseAuth *auth.Client
var flyApiToken string
var flyApp string
var flyApiUrl = "https://api.machines.dev/v1"

const addr = ":9090"

func init() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env variables")
	}

	flyApiToken = os.Getenv("FLY_API_TOKEN")
	flyApp = os.Getenv("FLY_APP")
	flyFirebaseCreds, err := storeSecretFirebaseCredsAsFile()

	if err != nil {
		log.Fatalf("Error storing Firebase credentials: %v", err)
	}
	var options []option.ClientOption

	options = append(options, option.WithCredentialsFile(flyFirebaseCreds))

	app, err := firebase.NewApp(ctx, nil, options...)
	if err != nil {
		log.Fatalf("Firebase initialization error: %v", err)
	}

	firebaseAuth, err = app.Auth(ctx)
	if err != nil {
		log.Fatalf("Firebase Auth initialization error: %v", err)
	}
}

// Middleware to verify Firebase JWT Token
// Middleware to verify Firebase JWT Token
func authMiddleware(c *fiber.Ctx) error {

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

// 🚀 Deploy Machine (Clone or New)
func deployMachine(c *fiber.Ctx) error {

	clone := c.Query("clone") == "true"
	masterId := c.Query("master_id")

	var config map[string]interface{}

	if clone && masterId != "" {
		machine, err := getMachineDetails(masterId)
		// log.Println("Machine:", machine)

		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		config = map[string]interface{}{
			"region": machine["region"],
			"config": machine["config"],
		}
	} else {
		log.Println("no Machine:", masterId)
		config = map[string]interface{}{
			"region": "ord",
			"config": map[string]interface{}{
				"image":    "registry.fly.io/deepcrawlqueue:deployment-01JMSKTJ0WQ92SHVZFP9REAWH7",
				"guest":    map[string]interface{}{"cpu_kind": "shared", "cpus": 2, "memory_mb": 2048},
				"services": []map[string]interface{}{{"internal_port": 8080, "protocol": "tcp"}},
			},
		}
	}

	// log.Println("Config:", config)

	machine, err := flyRequest("POST", fmt.Sprintf("%s/apps/%s/machines", flyApiUrl, flyApp), config)
	if err != nil {
		log.Fatalf("Fly Request Error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Machine deployed", "machine_id": machine["id"]})
}

// 🚀 Get Machine IP
func getMachineDetails(machineId string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/apps/%s/machines/%s", flyApiUrl, flyApp, machineId)
	log.Println("Get Machine url:", url)
	return flyRequest("GET", url, nil)
}

// 🚀 Start Machine
func startMachine(c *fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/start", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine started"})
}

// 🚀 Stop Machine
func stopMachine(c *fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/stop", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine stopped"})
}

// 🚀 Delete Machine
func deleteMachine(c *fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("DELETE", fmt.Sprintf("%s/apps/%s/machines/%s", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine deleted"})
}

// 🚀 Execute Task on Running Machine
func executeTask(c *fiber.Ctx) error {
	machineId := c.Params("machine_id")
	machine, err := getMachineDetails(machineId)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	machineIp := machine["private_ip"].(string)
	taskUrl := fmt.Sprintf("http://%s:8080/run-task", machineIp)

	_, err = flyRequest("POST", taskUrl, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Task executed on cloned machine"})
}

// 🚀 Helper Function for Fly.io API Requests
func flyRequest(method string, url string, body interface{}) (map[string]interface{}, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	req.Header.SetMethod(method)
	req.Header.Set("Authorization", "Bearer "+flyApiToken)
	req.Header.Set("Content-Type", "application/json")

	// Add URL to the request
	req.SetRequestURI(url)

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req.SetBody(jsonBody)
	}

	client := &fasthttp.Client{}
	err := client.Do(req, res)
	if err != nil {
		return nil, err
	}

	var responseData map[string]interface{}
	json.Unmarshal(res.Body(), &responseData)
	return responseData, nil
}

func storeSecretFirebaseCredsAsFile() (string, error) {
	// Get the secret from environment variables
	encodedCreds := os.Getenv("FIREBASE_CREDENTIALS")
	if encodedCreds == "" {
		_, err := os.Open(os.Getenv("FILE_FIREBASE_CREDENTIALS"))
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("File does not exist")
			} else {
				log.Fatal(err)
			}

			fmt.Println("FIREBASE_CREDENTIALS not set")
			return "", fmt.Errorf("FIREBASE_CREDENTIALS not set")
		}

		return os.Getenv("FILE_FIREBASE_CREDENTIALS"), nil
	}

	// Decode Base64
	decoded, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		fmt.Println("Error decoding Firebase credentials:", err)
		return "", fmt.Errorf("Error decoding Firebase credentials: %v", err)
	}

	// Save as a JSON file
	filePath := "/tmp/" + os.Getenv("FILE_FIREBASE_CREDENTIALS")
	err = os.WriteFile(filePath, decoded, 0644)
	if err != nil {
		fmt.Println("Error writing Firebase credentials file:", err)
		return "", fmt.Errorf("Error writing Firebase credentials file: %v", err)
	}

	fmt.Println("Firebase credentials saved at:", filePath)

	return filePath, nil

}

func Welcome(c *fiber.Ctx) error {
	return c.SendString("Welcome to the Deploy4Scrap API!")
}

func main() {
	// Initialize standard Go html template engine
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Prefork: true, // Enable prefork mode for better performance
		// Concurrency:    100,  // Set the desired concurrency level
		JSONEncoder:    json.Marshal,
		JSONDecoder:    json.Unmarshal,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		BodyLimit:      4 * 1024 * 1024, // 4 MB
		ReadBufferSize: 16 * 1024,       // 16 KB, or // 4 KB
		Views:          engine,          // Set View Engine
	})

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

	// Routes that don't require authentication
	app.Get("/", Welcome)

	// Start Metrics server
	app.Get("/metricsgraph", monitor.New(monitor.Config{Title: "Deploy4Scrap Metrics Page"}))

	// Create a group for authenticated routes
	authedApp := app.Group("/api", authMiddleware)

	authedApp.Post("/deploy", deployMachine)
	authedApp.Put("/machine/:id/start", startMachine)
	authedApp.Put("/machine/:id/stop", stopMachine)
	authedApp.Delete("/machine/:id", deleteMachine)
	// authedApp.Post("/execute-task/:machine_id", executeTask)

	// listener, err := reuseport.Listen("tcp4", "0.0.0.0"+addr)
	// if err != nil {
	// 	log.Fatalf("Failed to listen on port 3401 with SO_REUSEPORT: %v", err)
	// }
	// defer listener.Close()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Channel to listen for interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// go fly.WalkResponse()

	// Start Metrics server
	// go func() {
	// 	slog.Info("serving metrics", slog.String("addr", addr))
	// 	// http.Handle("/metrics", prometheus.Middleware())
	// 	// if err := http.Serve(listener, nil); err != nil {
	// 	// 	log.Fatal(err)
	// 	// }
	// }()

	// Start deploy4scrap MicroService
	go func() {
		log.Println("Starting deploy4scrap microservice on", os.Getenv("PORT"))
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Shutdown Fiber Server App
	if err := app.Shutdown(); err != nil {
		log.Fatal("Failed to shutdown server", zap.Error(err))
	}

	log.Println("deploy4scrap MicroService gracefully stopped")

	// Wait for an interrupt signal
	<-sigCh

	log.Println("Shutting metrics Server...")

}
