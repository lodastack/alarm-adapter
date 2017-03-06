package ping

import (
	"os"
	"time"

	"github.com/lodastack/alarm-adapter/config"

	"github.com/lodastack/log"
)

const PullInterval = 3

var Hostname string

func init() {
	var err error
	Hostname, err = os.Hostname()
	if err != nil {
		Hostname = "unknown"
	}
}

func Start() {
	if !config.C.Ping.Enable {
		log.Infof("ping module not enabled")
		return
	}
	r := NewRegistry(config.C.Main.RegistryAddr, config.C.Ping.IpList)
	m := NewPingMaster()
	workFunc := func() {
		servers, err := r.Servers()
		if err != nil {
			log.Errorf("get servers failed:%s", err)
		} else {
			go m.Do(servers)
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
