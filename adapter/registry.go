package adapter

import (
	"encoding/json"
	"fmt"

	"github.com/lodastack/alarm-adapter/requests"
	"github.com/lodastack/models"
)

// unit: min
const defaultPullInterval = 2

type Registry struct {
	Addr     string
	Interval int
}

type Resp struct {
	Status int            `json:"httpstatus"`
	Data   []models.Alarm `json:"data"`
}

func NewRegistry(addr string) *Registry {
	r := &Registry{
		Addr:     addr,
		Interval: defaultPullInterval,
	}
	return r
}

func (r *Registry) Alarms() (map[string]models.Alarm, error) {
	var resp Resp
	alarms := make(map[string]models.Alarm)
	url := fmt.Sprintf("%s/api/v1/agent/resource?ns=leaf.test.loda&type=alarm", r.Addr)
	response, err := requests.Get(url)
	if err != nil {
		return alarms, err
	}

	if response.Status == 200 {
		err = json.Unmarshal(response.Body, &resp)
		if err != nil {
			return alarms, err
		}
		for _, a := range resp.Data {
			alarms[a.Version] = a
		}
		return alarms, nil
	}

	return alarms, fmt.Errorf("get alarms failed: code %d", response.Status)
}
