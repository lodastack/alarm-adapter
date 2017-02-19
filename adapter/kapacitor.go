package adapter

import (
	"fmt"
	"time"

	"github.com/lodastack/log"
	"github.com/lodastack/models"

	"github.com/influxdata/kapacitor/client/v1"
)

const alertURL = "http://test.com/send"

type Kapacitor struct {
	Addrs   []string
	Clients map[string]*client.Client
	Hash    *Consistent
}

func NewKapacitor(addrs []string) *Kapacitor {
	c := NewConsistent()
	clients := make(map[string]*client.Client)
	for _, addr := range addrs {
		c.Add(addr)

		config := client.Config{
			URL:     addr,
			Timeout: time.Duration(3) * time.Second,
		}
		c, err := client.New(config)
		if err != nil {
			log.Errorf("new kapacitor %s client failed: %s", addr, err)
		}
		clients[addr] = c
	}

	k := &Kapacitor{
		Addrs:   addrs,
		Hash:    c,
		Clients: clients,
	}

	return k
}

func (k *Kapacitor) Tasks() map[string]client.Task {
	tasks := make(map[string]client.Task)
	for _, url := range k.Addrs {
		c, ok := k.Clients[url]
		if !ok {
			log.Errorf("get cache kapacitor %s client failed", url)
			continue
		}
		var listOpts client.ListTasksOptions
		listOpts.Default()
		ts, err := c.ListTasks(&listOpts)
		if err != nil {
			log.Errorf("list kapacitor %s client failed: %s", url, err)
			continue
		}
		for _, t := range ts {
			tasks[t.ID] = t
		}
	}
	return tasks
}

func (k *Kapacitor) Work(tasks map[string]client.Task, alarms map[string]models.Alarm) {
	for id, alarm := range alarms {
		if _, ok := tasks[id]; ok {
			continue
		}
		go k.CreateTask(alarm)
	}

	for id, task := range tasks {
		if _, ok := alarms[id]; ok {
			continue
		}
		go k.RemoveTask(task)
	}
}

// Create a new task.
// Errors if the task already exists.
func (k *Kapacitor) CreateTask(alarm models.Alarm) error {
	tick, err := GenTick(alarm)
	if err != nil {
		log.Errorf("gen tick script failed:%s", err)
		return err
	}
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
		ID:         alarm.Version,
		Type:       client.BatchTask,
		DBRPs:      dbrps,
		TICKscript: tick,
		Status:     status,
	}

	url := k.hashKapacitor(alarm.Version)
	c, ok := k.Clients[url]
	if !ok {
		log.Errorf("get cache kapacitor %s client failed", url)
		return fmt.Errorf("get cache kapacitor %s client failed", url)
	}
	log.Infof("create task:%s at %s", alarm.Version, url)
	_, err = c.CreateTask(createOpts)
	if err != nil {
		log.Errorf("create task at %s failed:%s", url, err)
	}
	return err
}

func (k *Kapacitor) RemoveTask(task client.Task) error {
	url := k.hashKapacitor(task.ID)
	c, ok := k.Clients[url]
	if !ok {
		log.Errorf("get cache kapacitor %s client failed", url)
		return fmt.Errorf("get cache kapacitor %s client failed", url)
	}
	return c.DeleteTask(c.TaskLink(task.ID))
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
        .post('%s')
        .slack()`
	res := fmt.Sprintf(batch, alarm.Func, alarm.DB, alarm.RP, alarm.Measurement,
		alarm.Period, alarm.Every, alarm.Func, alarm.Expression, alarm.Value, alertURL)
	return res, nil
}
