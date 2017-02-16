package adapter

import (
	"fmt"
	"time"

	"github.com/lodastack/alarm-adapter/config"
)

const defaultInterval = 1

func Start() {
	k := NewKapacitor(config.C.Main.KapacitorAddr)
	r := NewRegistry(config.C.Main.RegistryAddr)

	var ticker *time.Ticker
	duration := time.Duration(defaultInterval) * time.Minute
	ticker = time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			fmt.Printf("%v\n", k.Tasks())
			alarms, err := r.Alarms()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("%v\n", alarms)
		}
	}
}
