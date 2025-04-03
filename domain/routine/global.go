package routine

import "time"

type IGlobal interface {
	GetCurrentTime() time.Time
	GetMachineDetails(machineId, flyApiUrl, flyApp string) (map[string]interface{}, error)
	FlyRequest(method string, url string, body interface{}) (map[string]interface{}, error)
}
