package api

import (
	"encoding/json"
	"fmt"

	"github.com/lodastack/alarm-adapter/requests"
	"github.com/lodastack/log"
	"github.com/lodastack/models"
)

// unit: min
const defaultPullInterval = 2

type Registry struct {
	Addr     string
	NS       []string
	Global   bool
	Interval int
}

type RespNS struct {
	Status int      `json:"httpstatus"`
	Data   []string `json:"data"`
}

type RespCollect struct {
	Status int                 `json:"httpstatus"`
	Data   []map[string]string `json:"data"`
}

func NewRegistry(addr string, ns []string, g bool) *Registry {
	r := &Registry{
		Addr:     addr,
		NS:       ns,
		Global:   g,
		Interval: defaultPullInterval,
	}
	return r
}

func (r *Registry) NameSpaces() ([]string, error) {
	var resp RespNS
	var ns []string
	url := fmt.Sprintf("%s/api/v1/alarm/ns?format=list", r.Addr)
	response, err := requests.Get(url)
	if err != nil {
		return ns, err
	}

	if response.Status == 200 {
		err = json.Unmarshal(response.Body, &resp)
		if err != nil {
			return ns, err
		}
		return resp.Data, nil
	}
	return ns, fmt.Errorf("get all ns failed: code %d", response.Status)
}

func (r *Registry) APIs() (map[string][]models.HTTPResponse, error) {
	apisMap := make(map[string][]models.HTTPResponse)

	if r.Global {
		ns, err := r.NameSpaces()
		if err != nil {
			return apisMap, err
		}
		r.NS = ns
	}

	for _, leaf := range r.NS {
		var resp RespCollect
		var apis []models.HTTPResponse
		url := fmt.Sprintf("%s/api/v1/alarm/resource?ns=%s&type=collect", r.Addr, leaf)
		response, err := requests.Get(url)
		if err != nil {
			log.Errorf("get ns:%s machines failed: %s", leaf, err)
			continue
		}

		if response.Status == 200 {
			err = json.Unmarshal(response.Body, &resp)
			if err != nil {
				log.Errorf("unmarshal failed:%s", err)
				continue
			}
			for _, item := range resp.Data {
				monitorType, ok := item["measurement_type"]
				if !ok {
					log.Warning("measurement_type is not exist: ", item["measurement_type"])
					continue
				}
				if monitorType == "API" {
					b, err := json.Marshal(item)
					if err != nil {
						log.Warning("json.Marshal item failed: ", err)
						continue
					}
					var api models.HTTPResponse
					err = json.Unmarshal(b, &api)
					apis = append(apis, api)
				}
			}
			apisMap[leaf] = apis
		} else {
			log.Errorf("get ns:%s collects http failed: status %d", leaf, response.Status)
		}
	}
	return apisMap, nil
}
