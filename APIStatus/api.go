package api

import (
	"github.com/lodastack/alarm-adapter/config"
	"time"

	"github.com/lodastack/log"
)

const PullInterval = 3

func Start() {
	if !config.C.API.Enable {
		log.Infof("monitor API module not enabled")
		return
	}
	r := NewRegistry(config.C.Main.RegistryAddr)
	m := NewAPIMaster()

	workFunc := func() {
		apis, err := r.APIs()
		if err != nil {
			log.Errorf("get api collect failed:%s", err)
		} else {
			go m.Do(apis)
		}
	}
	workFunc()
	ticker := time.NewTicker(time.Duration(PullInterval) * time.Minute)
	for {
		select {
		case <-ticker.C:
			workFunc()
		}
	}
}
