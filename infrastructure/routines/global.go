package routines

import (
	"fmt"
	"log"
	"time"

	"github.com/goccy/go-json"

	"github.com/valyala/fasthttp"
)

type Global struct {
	// encrypted *domain.EncryptedData
	// security *secrets.Security
	// logger   *zap.Logger
	flyApiToken string
}

func NewGlobalRoutines(flyApiToken string /* logger *zap.Logger */) *Global {
	return &Global{flyApiToken /* security: secrets.NewSecurity(logger), logger: logger */}
}

func (g *Global) GetCurrentTime() time.Time {
	return time.Now()
}

// 🚀 Get Machine IP
func (g *Global) GetMachineDetails(machineId, flyApiUrl, flyApp string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/apps/%s/machines/%s", flyApiUrl, flyApp, machineId)
	log.Println("Get Machine url:", url)
	return g.FlyRequest("GET", url, nil)
}

// 🚀 Helper Function for Fly.io API Requests
func (g *Global) FlyRequest(method string, url string, body interface{}) (map[string]interface{}, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	req.Header.SetMethod(method)
	req.Header.Set("Authorization", "Bearer "+g.flyApiToken)
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
	err = json.Unmarshal(res.Body(), &responseData)
	if err != nil {
		// Handle the error appropriately, e.g., log it or return an error response
		return nil, err
	}
	return responseData, nil
}
