package snmp

import (
	"os"
	"time"

	"github.com/lodastack/alarm-adapter/config"
	"github.com/lodastack/log"
)

const PullInterval = 3

var Hostname string

func init() {
	historymap = make(map[string]int64)
	var err error
	Hostname, err = os.Hostname()
	if err != nil {
		Hostname = "unknown"
	}
}

func Start() {
	if !config.C.SNMP.Enable {
		log.Infof("monitor SNMP module not enabled")
		return
	}
	r := NewRegistry(config.C.Main.RegistryAddr, config.C.SNMP.NS, config.C.SNMP.IpList)
	m := NewSnmpMaster(config.C.SNMP.Community)
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
