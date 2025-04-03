package handlers

import (
	"fmt"
	"log"

	"github.com/AntoniadisCorp/deploy4scrap/domain"
	"github.com/AntoniadisCorp/deploy4scrap/domain/routine"
	"github.com/AntoniadisCorp/deploy4scrap/infrastructure/routines"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	globalRoutines routine.IGlobal
	// validator *validator.CustomValidator

	// AUTH_SESSION_NAME          string
	// OAUTH_STATE_COOKIE         string
	// OAUTH_CODE_VERIFIER_COOKIE string
	// OAUTH_CODE_CHAL_COOKIE     string
	// CSRF_SESSION_NAME          string

	// cfg    *config.Config
	// logger *zap.Logger
	flyApp    string
	flyApiUrl string
}

func NewHandlers(flyApiToken, flyApiUrl, flyApp string) *AuthHandler {

	return &AuthHandler{
		globalRoutines: routines.NewGlobalRoutines(flyApiToken),
		flyApp:         flyApp,
		flyApiUrl:      flyApiUrl,
		// cfg:                        cfg,
		// logger:                     logger,
	}
}

// 🚀 Deploy Machine (Clone or New)
func (h *AuthHandler) DeployMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {

	clone := c.Query("clone") == "true"
	masterId := c.Query("master_id")

	var config map[string]interface{}
	var response domain.HTTPResponse

	if clone && masterId != "" {
		machine, err := h.globalRoutines.GetMachineDetails(masterId, h.flyApiUrl, h.flyApp)
		// log.Println("Machine:", machine)

		if err != nil {
			response = domain.HTTPResponse{
				// Headers: headers,
				Code:    fiber.StatusBadRequest,
				Status:  "error",
				Message: "Check the Machine Details again",
				Errors:  domain.APIError{Code: "Get Machine Details Failed", Message: err.Error()},
				Data:    fiber.Map{"status": false},
			}

			return &response, err
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
	machine, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines", h.flyApiUrl, h.flyApp), config)
	if err != nil {
		log.Fatalf("Fly Request Error: %v", err)
		response = domain.HTTPResponse{
			// Headers: headers,
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Check the Fly api url or fly app image if exists",
			Errors:  domain.APIError{Code: "Deploy by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}

	response = domain.HTTPResponse{
		// Headers: headers,
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine deployed Successful",
		Data:    fiber.Map{"status": true, "message": "Machine deployed", "machine_id": machine["id"]},
	}

	return &response, nil

}

// 🚀 Start Machine
func (h *AuthHandler) StartMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse

	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}

	// post request to start machine by id
	_, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/start", h.flyApiUrl, h.flyApp, machineId), nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to start machine",
			Errors:  domain.APIError{Code: "StartMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}
	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine started",
		Data:    fiber.Map{"status": true, "message": "Machine started"},
	}
	return &response, nil
}

// 🚀 Stop Machine
func (h *AuthHandler) StopMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse

	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}
	_, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/stop", h.flyApiUrl, h.flyApp, machineId), nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to stop the machine by id " + machineId,
			Errors:  domain.APIError{Code: "StopMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}

	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine stopped",
		Data:    fiber.Map{"status": true, "message": "Machine stopped"},
	}
	return &response, nil
}

// 🚀 Delete Machine
func (h *AuthHandler) DeleteMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse
	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}
	_, err := h.globalRoutines.FlyRequest("DELETE", fmt.Sprintf("%s/apps/%s/machines/%s", h.flyApiUrl, h.flyApp, machineId), nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete the machine by id " + machineId,
			Errors:  domain.APIError{Code: "DeleteMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}
	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine deleted",
		Data:    fiber.Map{"status": true, "message": "Machine deleted"},
	}
	return &response, nil
}

// 🚀 Execute Task on Running Machine
func (h *AuthHandler) ExecuteTask(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("machine_id")
	var response domain.HTTPResponse

	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}

	machine, err := h.globalRoutines.GetMachineDetails(machineId, h.flyApiUrl, h.flyApp)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get machine details",
			Errors:  domain.APIError{Code: "GetMachineDetails by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	machineIp := machine["private_ip"].(string)
	taskUrl := fmt.Sprintf("http://%s:8080/run-task", machineIp)

	_, err = h.globalRoutines.FlyRequest("POST", taskUrl, nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to execute task on machine",
			Errors:  domain.APIError{Code: "ExecuteTask by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Task executed on machine",
		Data:    fiber.Map{"status": true, "message": "Task executed on machine"},
	}
	return &response, nil
}

func machineIdRequired() domain.HTTPResponse {

	return domain.HTTPResponse{
		Code:    fiber.StatusBadRequest,
		Status:  "error",
		Message: "Machine ID is required",
		Errors:  domain.APIError{Code: "Machine ID is required", Message: "Machine ID is required"},
		Data:    fiber.Map{"status": false},
	}
}
