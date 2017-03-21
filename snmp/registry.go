package snmp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lodastack/alarm-adapter/requests"
	"github.com/lodastack/log"
)

// unit: min
const defaultPullInterval = 2

const spliter = ","

type Registry struct {
	Addr     string
	NS       []string
	Allow    []string
	Interval int
}

type RespNS struct {
	Status int      `json:"httpstatus"`
	Data   []string `json:"data"`
}

type RespMachine struct {
	Status int       `json:"httpstatus"`
	Data   []Machine `json:"data"`
}

type Machine struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

type Server struct {
	IP       []string `json:"ip"`
	Hostname string   `json:"hostname"`
}

func NewRegistry(addr string, ns []string, allow []string) *Registry {
	r := &Registry{
		Addr:     addr,
		NS:       ns,
		Allow:    allow,
		Interval: defaultPullInterval,
	}
	return r
}

func (r *Registry) NameSpaces() []string {
	var ns []string
	for _, leaf := range r.NS {
		var resp RespNS
		url := fmt.Sprintf("%s/api/v1/alarm/ns?ns=%s&format=list", r.Addr, leaf)
		response, err := requests.Get(url)
		if err != nil {
			log.Errorf("get ns failed: %s", err)
			continue
		}

		if response.Status == 200 {
			err = json.Unmarshal(response.Body, &resp)
			if err != nil {
				log.Errorf("unmarshal ns body failed: %s", err)
				continue
			}
			ns = append(ns, resp.Data...)
		}
	}
	return ns
}

func (r *Registry) Servers() (map[string][]Server, error) {
	serversMap := make(map[string][]Server)
	for _, leaf := range r.NameSpaces() {
		leaf := leaf + ".loda"
		var resp RespMachine
		var servers []Server
		url := fmt.Sprintf("%s/api/v1/alarm/resource?ns=%s&type=machine", r.Addr, leaf)
		response, err := requests.Get(url)
		if err != nil {
			log.Errorf("get ns:%s machines failed: %s", leaf, err)
			continue
		}

		if response.Status == 200 {
			err = json.Unmarshal(response.Body, &resp)
			if err != nil {
				log.Errorf("unmarshal failed:%s", err)
				continue
			}
			for _, a := range resp.Data {
				var s Server
				s.IP = r.allow(a.IP)
				s.Hostname = a.Hostname
				servers = append(servers, s)
			}
			serversMap[leaf] = servers
		} else {
			log.Errorf("get ns:%s machines http failed: status %d", leaf, response.Status)
		}
	}
	return serversMap, nil
}

func (r *Registry) allow(ipStr string) []string {
	var res []string
	monited := false
	ips := strings.Split(ipStr, spliter)
	for _, ip := range ips {
		for _, set := range r.Allow {
			matched, _ := regexp.MatchString(set, ip)
			if matched {
				res = append(res, ip)
				monited = true
				break
			}
		}
		if !monited {
			log.Infof("Don't monitor IP:%s", ip)
		}
		monited = false
	}
	return res
}
