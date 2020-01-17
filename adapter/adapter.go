package adapter

import (
	"time"

	"github.com/lodastack/alarm-adapter/config"

	"github.com/lodastack/log"
)

const defaultInterval = 1
const updateInterval = 3

func Start() {
	if !config.C.Alarm.Enable {
		log.Infof("alarm module not enabled")
		return
	}
	r := NewRegistry(config.C.Main.RegistryAddr, config.C.Alarm.NS)
	servers, err := r.AlarmServers()
	if err != nil {
		panic(err)
	}
	k := NewKapacitor(servers, config.C.Alarm.EventAddr)

	go updateAlarmServers(k, r)

	for {
	SLEEP:
		time.Sleep(time.Duration(defaultInterval) * time.Minute)

		tasks, err := k.Tasks()
		if err != nil || tasks == nil {
			log.Errorf("get tasks failed: %v", err)
			goto SLEEP
		}
		alarms, err := r.Alarms()
		if err != nil || alarms == nil || len(alarms) == 0 {
			log.Errorf("get alarms failed: %v", err)
			goto SLEEP
		}
		k.Work(tasks, alarms)
	}
}

func updateAlarmServers(k *Kapacitor, r *Registry) {
	ticker := time.NewTicker(time.Duration(updateInterval) * time.Minute)
	for {
		select {
		case <-ticker.C:
			servers, err := r.AlarmServers()
			if err == nil {
				k.SetAddr(servers)
			} else {
				log.Error(err)
			}
		}
	}
}
