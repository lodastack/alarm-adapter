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
