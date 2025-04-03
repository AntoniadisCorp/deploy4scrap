package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	router "github.com/AntoniadisCorp/deploy4scrap/interfaces/controller"
	"github.com/AntoniadisCorp/deploy4scrap/interfaces/handlers"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Define a struct to hold the query parameters
type DeployQuery struct {
	Clone    bool   `query:"clone"`
	MasterID string `query:"master_id"`
}

var firebaseAuth *auth.Client
var FirebaseApp *firebase.App
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

	FirebaseApp, err = firebase.NewApp(ctx, nil, options...)
	if err != nil {
		log.Fatalf("Firebase initialization error: %v", err)
	}
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
	err = os.WriteFile(filePath, decoded, 0600)
	if err != nil {
		fmt.Println("Error writing Firebase credentials file:", err)
		return "", fmt.Errorf("Error writing Firebase credentials file: %v", err)
	}

	fmt.Println("Firebase credentials saved at:", filePath)

	return filePath, nil

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

	// Initialize Handlers
	handler := handlers.NewHandlers(flyApiToken, flyApiUrl, flyApp)

	// Setup routes
	router.SetupRoutes(app, handler, FirebaseApp)
	// Setup Non-Accessible routes
	router.SetupNonAccessibleRoutes(app)

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
