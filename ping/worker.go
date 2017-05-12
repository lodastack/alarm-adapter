package ping

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/lodastack/log"
	"github.com/lodastack/models"
	"github.com/lodastack/sdk-go"
)

// unit: second
const DefaultPingInterval = 30

// unit: second
// individual target initial timeout
const DefaultTimeout = 1

type PingMaster struct {
	workers map[string]*PingWorker
	mu      sync.RWMutex

	countmu sync.RWMutex
	count   int
}

func NewPingMaster() *PingMaster {
	return &PingMaster{
		workers: make(map[string]*PingWorker),
		count:   0,
	}
}

func (master *PingMaster) Do(m map[string][]Server) {
	//lock all
	master.mu.Lock()
	defer master.mu.Unlock()

	//check create ping worker
	for ns, servers := range m {
		for _, server := range servers {
			for _, ip := range server.IP {
				if _, ok := master.workers[ns+ip]; !ok {
					err := master.CreateWorker(ns, ip, server.Hostname)
					if err != nil {
						log.Errorf("master create worker failed: %s", err)
					}
				}
			}
		}
	}
	// check remove ping worker
	for name := range master.workers {
		if serverExist(name, m) {
			continue
		}
		err := master.RemoveWorker(name)
		if err != nil {
			log.Errorf("master remove worker failed: %s", err)
		}
	}
}

// thread unsafe
func (master *PingMaster) CreateWorker(ns string, ip string, hostname string) error {
	w := NewPingWorker(ns, ip, hostname)
	go w.Run()
	master.countmu.Lock()
	master.count++
	master.countmu.Unlock()
	master.workers[ns+ip] = w
	return nil
}

// thread unsafe
func (master *PingMaster) RemoveWorker(name string) error {
	if w, ok := master.workers[name]; ok && w != nil {
		w.Stop()
		master.countmu.Lock()
		master.count--
		master.countmu.Unlock()
		master.workers[name] = nil
		delete(master.workers, name)
	}
	return nil
}

type PingWorker struct {
	ns       string
	ip       string
	hostname string
	interval int
	done     chan struct{}
	opened   bool
}

func NewPingWorker(ns string, ip string, hostname string) *PingWorker {
	return &PingWorker{
		ns:       ns,
		ip:       ip,
		hostname: hostname,
		interval: DefaultPingInterval,
		done:     make(chan struct{}),
		opened:   false,
	}
}

func (w *PingWorker) Run() {
	if w.opened {
		return
	}
	w.opened = true
	// LB all ping tasks
	randNum := rand.Intn(DefaultPingInterval * 1000)
	time.Sleep(time.Duration(randNum) * time.Millisecond)

	pingfunc := func() {
		loss := Ping(w.ip, DefaultTimeout)
		go Send(w.ns, w.hostname, w.ip, loss)
		log.Debugf("Ping [%s] %s  %s Loss:%v", w.ns, w.ip, w.hostname, loss)
	}

	pingfunc()
	ticker := time.NewTicker(time.Duration(DefaultPingInterval) * time.Second)
	for {
		select {
		case <-ticker.C:
			pingfunc()
		case <-w.done:
			goto exit
		}
	}

exit:
	log.Infof("Ping Worker %s exit", w.ns+w.ip)
}

func (w *PingWorker) Stop() {
	if !w.opened {
		return
	}
	w.opened = false
	close(w.done)
}

func serverExist(name string, m map[string][]Server) bool {
	for ns, servers := range m {
		for _, server := range servers {
			for _, ip := range server.IP {
				if name == ns+ip {
					return true
				}
			}
		}
	}
	return false
}

func Send(ns string, hostname string, ip string, loss float64) error {
	m := models.Metric{
		Name:      "ping.loss",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"ip":   ip,
			"host": hostname,
		},
		Value: loss,
	}
	data, err := json.Marshal([]models.Metric{m})
	if err != nil {
		return err
	}
	return sdk.Post(ns, data)
}
