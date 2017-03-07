package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lodastack/log"
	"github.com/lodastack/models"
	"github.com/lodastack/sdk-go"
)

// unit: second
const DefaultMonitorInterval = 60

// unit: second
// individual target initial timeout
const DefaultTimeout = 3

type APIMaster struct {
	workers map[string]*APIWorker
	mu      sync.RWMutex

	countmu sync.RWMutex
	count   int
}

func NewAPIMaster() *APIMaster {
	return &APIMaster{
		workers: make(map[string]*APIWorker),
		count:   0,
	}
}

func (master *APIMaster) Do(m map[string][]models.HTTPResponse) {
	//lock all
	master.mu.Lock()
	defer master.mu.Unlock()

	//check create api monitor worker
	for ns, apis := range m {
		for _, api := range apis {
			if _, ok := master.workers[ns+api.Name]; !ok {
				err := master.CreateWorker(ns, api)
				if err != nil {
					log.Errorf("master create worker failed: %s", err)
				}
			}
		}
	}
	// check remove api monitor worker
	for name := range master.workers {
		if apiExist(name, m) {
			continue
		}
		err := master.RemoveWorker(name)
		if err != nil {
			log.Errorf("master remove worker failed: %s", err)
		}
	}
}

func apiExist(name string, m map[string][]models.HTTPResponse) bool {
	for ns, apis := range m {
		for _, api := range apis {
			if name == ns+api.Name {
				return true
			}
		}
	}
	return false
}

// thread unsafe
func (master *APIMaster) CreateWorker(ns string, m models.HTTPResponse) error {
	w := NewAPIWorker(ns, m)
	go w.Run()
	master.countmu.Lock()
	master.count++
	master.countmu.Unlock()
	master.workers[ns+m.Name] = w
	return nil
}

// thread unsafe
func (master *APIMaster) RemoveWorker(name string) error {
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

type APIWorker struct {
	ns       string
	meta     models.HTTPResponse
	interval int
	done     chan struct{}
	opened   bool
}

func NewAPIWorker(ns string, m models.HTTPResponse) *APIWorker {
	return &APIWorker{
		ns:       ns,
		meta:     m,
		interval: DefaultMonitorInterval,
		done:     make(chan struct{}),
		opened:   false,
	}
}

func (w *APIWorker) Run() {
	if w.opened {
		return
	}
	w.opened = true
	// LB all api monitor tasks
	randNum := rand.Intn(DefaultMonitorInterval * 1000)
	time.Sleep(time.Duration(randNum) * time.Millisecond)
	monitorAPI(w.ns, w.meta)
	ticker := time.NewTicker(time.Duration(w.interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			go monitorAPI(w.ns, w.meta)
		case <-w.done:
			goto exit
		}
	}

exit:
	log.Infof("api monitor Worker %s exit", w.ns+w.meta.Name)
}

func (w *APIWorker) Stop() {
	if !w.opened {
		return
	}
	w.opened = false
	close(w.done)
}

func monitorAPI(ns string, h models.HTTPResponse) {
	fields := make(map[string]float64)
	fields["alive"] = 0
	defer func() {
		go Send(ns, h.Name, fields)
		log.Infof("API Monitor [%s] : %v", h.Address, fields)
	}()

	var interval int
	var err error
	if interval, err = strconv.Atoi(h.ResponseTimeout); err != nil {
		log.Errorf("convert string to int failed：%s", err)
		interval = DefaultTimeout
	}
	if interval < DefaultTimeout {
		interval = DefaultTimeout
	}
	tr := &http.Transport{
		ResponseHeaderTimeout: time.Duration(interval) * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(interval) * time.Second,
	}

	var body io.Reader
	if h.Body != "" {
		body = strings.NewReader(h.Body)
	}
	request, err := http.NewRequest(h.Method, h.Address, body)
	if err != nil {
		log.Errorf("API status new request Failed:%s", err)
		return
	}

	// Start Timer
	start := time.Now()
	resp, err := client.Do(request)
	if err != nil {
		log.Errorf("HTTP do failed:%s", err)
		return
	}
	fields["responseTime"] = time.Since(start).Seconds()
	fields["responseCode"] = float64(resp.StatusCode)

	// Check the response for status code.
	if resp.StatusCode/100 == 2 {
		fields["alive"] = 1
	}

	// Check the response for a regex match.
	if h.ResponseStringMatch != "" {

		// Compile once and reuse
		if h.CompiledStringMatch == nil {
			h.CompiledStringMatch = regexp.MustCompile(h.ResponseStringMatch)
			if err != nil {
				log.Errorf("Failed to compile regular expression %s : %s", h.ResponseStringMatch, err)
				fields["responseMatch"] = 0
				return
			}
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read body of HTTP Response : %s", err)
			fields["responseMatch"] = 0
			return
		}

		if h.CompiledStringMatch.Match(bodyBytes) {
			fields["responseMatch"] = 1
		} else {
			fields["responseMatch"] = 0
		}
	}
	return
}

func Send(ns string, name string, fields map[string]float64) error {
	var ms []models.Metric
	for k, v := range fields {
		m := models.Metric{
			Name:      name + "." + k,
			Timestamp: time.Now().Unix(),
			Value:     v,
		}
		ms = append(ms, m)
	}
	data, err := json.Marshal(ms)
	if err != nil {
		return err
	}
	return sdk.Post(ns, data)
}
