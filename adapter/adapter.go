package adapter

import (
	"time"

	"github.com/lodastack/alarm-adapter/config"

	"github.com/lodastack/log"
)

const defaultInterval = 1

func Start() {
	k := NewKapacitor(config.C.Main.KapacitorAddr)
	r := NewRegistry(config.C.Main.RegistryAddr)

	ticker := time.NewTicker(time.Duration(defaultInterval) * time.Minute)
	for {
		select {
		case <-ticker.C:
			tasks := k.Tasks()
			alarms, err := r.Alarms()
			if err != nil {
				log.Errorf("get alarms failed:%s", err)
			} else {
				go k.Work(tasks, alarms)
			}
		}
	}
}
