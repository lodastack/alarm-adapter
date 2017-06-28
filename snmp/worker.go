package snmp

import (
	"math/rand"
	"sync"
	"time"

	"github.com/lodastack/log"
)

// unit: second
const DefaultInterval = 60

type SnmpMaster struct {
	Community []string
	workers   map[string]*SnmpWorker
	mu        sync.RWMutex

	countmu sync.RWMutex
	count   int
}

func NewSnmpMaster(c []string) *SnmpMaster {
	return &SnmpMaster{
		Community: c,
		workers:   make(map[string]*SnmpWorker),
		count:     0,
	}
}

func (master *SnmpMaster) Do(m map[string][]Server) {
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
func (master *SnmpMaster) CreateWorker(ns string, ip string, hostname string) error {
	w := NewSnmpWorker(ns, ip, hostname, master.Community)
	go w.Run()
	master.countmu.Lock()
	master.count++
	master.countmu.Unlock()
	master.workers[ns+ip] = w
	return nil
}

// thread unsafe
func (master *SnmpMaster) RemoveWorker(name string) error {
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

type SnmpWorker struct {
	ns        string
	ip        string
	hostname  string
	community []string
	interval  int
	done      chan struct{}
	opened    bool
}

func NewSnmpWorker(ns string, ip string, hostname string, c []string) *SnmpWorker {
	return &SnmpWorker{
		ns:        ns,
		ip:        ip,
		hostname:  hostname,
		community: c,
		interval:  DefaultInterval,
		done:      make(chan struct{}),
		opened:    false,
	}
}

func (w *SnmpWorker) Run() {
	if w.opened {
		return
	}
	w.opened = true
	// LB all snmp tasks
	randNum := rand.Intn(DefaultInterval * 1000)
	time.Sleep(time.Duration(randNum) * time.Millisecond)

	snmpfunc := func() {
		traffic(w.ns, w.ip, w.hostname, w.community)
	}

	snmpfunc()
	ticker := time.NewTicker(time.Duration(DefaultInterval) * time.Second)
	for {
		select {
		case <-ticker.C:
			snmpfunc()
		case <-w.done:
			goto exit
		}
	}

exit:
	log.Infof("SNMP Worker %s exit", w.ns+w.ip)
}

func (w *SnmpWorker) Stop() {
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
