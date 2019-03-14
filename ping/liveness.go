package ping

import (
	"time"

	"github.com/lodastack/alarm-adapter/config"
)

// PingLive is a liveness of this service
var PingLive bool = true

func init() {
	go livenessTask()
}

func livenessTask() {
	ticker := time.NewTicker(time.Duration(10) * time.Second)
	if !config.C.Ping.Enable && len(config.C.Ping.Notary) == 0 {
		return
	}

	for {
		select {
		case <-ticker.C:
			publicPing()
		}
	}
}

func publicPing() {
	var loss float64
	for _, ip := range config.C.Ping.Notary {
		loss += Ping(ip, 2)
	}
	loss = loss / float64(len(config.C.Ping.Notary))
	if loss > 60 {
		PingLive = false
	} else {
		PingLive = true
	}
}
