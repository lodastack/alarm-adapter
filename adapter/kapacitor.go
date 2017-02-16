package adapter

import (
	"fmt"
	"time"

	"github.com/lodastack/alarm-adapter/config"
	"github.com/lodastack/log"
	"github.com/lodastack/models"

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

// Create a new task.
// Errors if the task already exists.
func (k *Kapacitor) CreateTask(alarm models.Alarm) error {
	tick, err := GenTick(alarm)
	dbrps := []client.DBRP{
		{
			Database:        alarm.DB,
			RetentionPolicy: alarm.RP,
		},
	}
	status := client.Disabled
	if alarm.Enable == "true" {
		status = client.Enabled
	}

	createOpts := client.CreateTaskOptions{
		ID:         alarm.ID,
		Type:       client.BatchTask,
		DBRPs:      dbrps,
		TICKscript: tick,
		Status:     status,
	}

	config := client.Config{
		URL:     config.C.Main.RegistryAddr,
		Timeout: time.Duration(3) * time.Second,
	}
	c, err := client.New(config)
	if err != nil {
		log.Errorf("new kapacitor %s client failed: %s", "url", err)
	}

	_, err = c.CreateTask(createOpts)
	return err
}

func GenTick(alarm models.Alarm) (string, error) {
	batch := `
batch
    |query('''
        SELECT %s(value)
        FROM "%s"."%s"."%s"
    ''')
        .period(%s)
        .every(%s)
        .groupBy(time(1m),'host')
    |alert()
        .crit(lambda: "mean" > %s)
        .slack()`
	res := fmt.Sprintf(batch, alarm.Func, alarm.DB, alarm.RP, alarm.Measurement, alarm.Period, alarm.Every, alarm.Value)
	return res, nil
}
