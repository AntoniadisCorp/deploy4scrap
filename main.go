package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/AntoniadisCorp/deploy4scrap/fly"
	"github.com/gofiber/contrib/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
func authMiddleware(c fiber.Ctx) error {

	// Ignore authentication for / and /metrics paths
	if c.Path() == "/" || c.Path() == "/metrics" {
		return c.Next()
	}

	token := c.Get("Authorization")
	token = token[len("Bearer "):]

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
func deployMachine(c fiber.Ctx) error {

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
func startMachine(c fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/start", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine started"})
}

// 🚀 Stop Machine
func stopMachine(c fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/stop", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine stopped"})
}

// 🚀 Delete Machine
func deleteMachine(c fiber.Ctx) error {
	machineId := c.Params("id")
	_, err := flyRequest("DELETE", fmt.Sprintf("%s/apps/%s/machines/%s", flyApiUrl, flyApp, machineId), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Machine deleted"})
}

// 🚀 Execute Task on Running Machine
func executeTask(c fiber.Ctx) error {
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

func Welcome(c fiber.Ctx) error {
	return c.SendString("Welcome to the Deploy4Scrap API!")
}

func main() {
	app := fiber.New()

	// Create a group for authenticated routes
	authedApp := app.Group("/", authMiddleware)

	authedApp.Post("/deploy", deployMachine)
	authedApp.Put("/machine/:id/start", startMachine)
	authedApp.Put("/machine/:id/stop", stopMachine)
	authedApp.Delete("/machine/:id", deleteMachine)
	// authedApp.Post("/execute-task/:machine_id", executeTask)

	// Routes that don't require authentication
	app.Get("/", Welcome)

	// Start Metrics server
	app.Get("/metrics", monitor.New(monitor.Config{Title: "Deploy4Scrap Metrics Page"}))

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Channel to listen for interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go fly.Walk()

	// Start Metrics server
	go func() {
		slog.Info("serving metrics", slog.String("addr", addr))
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Start deploy4scrap MicroService
	go func() {
		log.Println("Starting deploy4scrap microservice on ", zap.String("port", os.Getenv("PORT")))
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		log.Fatal("Failed to shutdown server", zap.Error(err))
	}

	log.Println("deploy4scrap MicroService gracefully stopped")

	// Wait for an interrupt signal
	<-sigCh

	log.Println("Shutting metrics Server...")

}
