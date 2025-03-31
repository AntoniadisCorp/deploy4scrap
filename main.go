package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
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

func init() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env variables")
	}

	flyApiToken = os.Getenv("FLY_API_TOKEN")
	flyApp = os.Getenv("FLY_APP")

	var options []option.ClientOption

	options = append(options, option.WithCredentialsFile("libnet-d76db-949683c2222d.json"))

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
func authMiddleware(c *fiber.Ctx) error {
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
func deployMachine(c *fiber.Ctx) error {

	clone := c.Query("clone") == "true"
	masterId := c.Query("master_id")

	var config map[string]interface{}

	if clone && masterId != "" {
		machine, err := getMachineDetails(masterId)
		log.Println("Machine:", machine)

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

	log.Println("Config:", config)

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

func main() {
	app := fiber.New()

	app.Use(authMiddleware)

	app.Post("/deploy", deployMachine)
	app.Put("/machine/:id/start", startMachine)
	app.Put("/machine/:id/stop", stopMachine)
	app.Delete("/machine/:id", deleteMachine)
	// app.Post("/execute-task/:machine_id", executeTask)

	log.Fatal(app.Listen(":" + os.Getenv("PORT")))
}
