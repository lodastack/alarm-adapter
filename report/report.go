package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/lodastack/alarm-adapter/config"

	"github.com/lodastack/log"
	"github.com/lodastack/models"
)

func Report(hostname string, ip string, ns string) {
	data := models.Report{
		NewIPList:   []string{ip},
		Ns:          []string{ns},
		Version:     "0.0.0",
		GoVersion:   runtime.Version(),
		NewHostname: hostname,
		OldHostname: hostname,
		AgentType:   "alarm-adapter",
		Update:      false,
		UpdateTime:  time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("json.Marshal failed: ", data)
	} else {
		url := fmt.Sprintf("%s/api/v1/agent/report", config.C.Main.RegistryAddr)
		resp, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Error("report agent info failed: ", err)
		} else {
			if resp.StatusCode == http.StatusOK {
				log.Infof("report agent %s info successfully", ip)
			} else {
				log.Errorf("report agent info failed: StatusCode %d", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}
}
