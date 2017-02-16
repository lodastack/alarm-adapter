package adapter

import (
	"time"

	"github.com/lodastack/log"

	"github.com/influxdata/kapacitor/client/v1"
)

type Kapacitor struct {
	Addrs []string
}

func NewKapacitor(addrs []string) *Kapacitor {
	k := &Kapacitor{
		Addrs: addrs,
	}
	return k
}

func (k *Kapacitor) Tasks() []client.Task {
	var tasks []client.Task
	for _, url := range k.Addrs {
		config := client.Config{
			URL:     url,
			Timeout: time.Duration(3) * time.Second,
		}
		c, err := client.New(config)
		if err != nil {
			log.Errorf("new kapacitor %s client failed: %s", url, err)
		}
		var listOpts client.ListTasksOptions
		listOpts.Default()
		t, err := c.ListTasks(&listOpts)
		if err != nil {
			log.Errorf("list kapacitor %s client failed: %s", url, err)
		} else {
			tasks = append(tasks, t...)
		}
	}
	return tasks
}
