package adapter

import (
	"fmt"
	"time"

	"github.com/lodastack/log"
	"github.com/lodastack/models"

	"github.com/influxdata/kapacitor/client/v1"
)

const alertURL = "http://api.msg.ifengidc.com:7989/sendRTX"

type Kapacitor struct {
	Addrs []string
	Hash  *Consistent
}

func NewKapacitor(addrs []string) *Kapacitor {
	c := NewConsistent()
	for _, addr := range addrs {
		c.Add(addr)
	}

	k := &Kapacitor{
		Addrs: addrs,
		Hash:  c,
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
		URL:     k.hashKapacitor(alarm.ID),
		Timeout: time.Duration(3) * time.Second,
	}
	c, err := client.New(config)
	if err != nil {
		log.Errorf("new kapacitor %s client failed: %s", "url", err)
	}

	_, err = c.CreateTask(createOpts)
	return err
}

func (k *Kapacitor) hashKapacitor(id string) string {
	choose, err := k.Hash.Get(id)
	if err != nil {
		log.Errorf("hash get server failed:%s", err)
		// need fix here, panic risk
		return k.Addrs[len(k.Addrs)-1]
	}
	return choose
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
        .crit(lambda: "%s" %s %s)
        .post(%s)
        .slack()`
	res := fmt.Sprintf(batch, alarm.Func, alarm.DB, alarm.RP, alarm.Measurement,
		alarm.Period, alarm.Every, alarm.Func, alarm.Expression, alarm.Value, alertURL)
	return res, nil
}
