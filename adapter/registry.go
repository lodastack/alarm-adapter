package adapter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lodastack/alarm-adapter/requests"
	"github.com/lodastack/models"
)

// unit: min
const defaultPullInterval = 2

// Regular expression to match intranet IP Address
// include: 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16
const REGIntrannetIP = `^((192\.168|172\.([1][6-9]|[2]\d|3[01]))(\.([2][0-4]\d|[2][5][0-5]|[01]?\d?\d)){2}|10(\.([2][0-4]\d|[2][5][0-5]|[01]?\d?\d)){3})$`

type Registry struct {
	Addr     string
	AlarmNS  string
	Interval int
}

type RespAlarm struct {
	Status int            `json:"httpstatus"`
	Data   []models.Alarm `json:"data"`
}

type RespMachine struct {
	Status int       `json:"httpstatus"`
	Data   []Machine `json:"data"`
}

type Machine struct {
	IP string `json:"ip"`
}

func NewRegistry(addr string, alarmNS string) *Registry {
	r := &Registry{
		Addr:     addr,
		AlarmNS:  alarmNS,
		Interval: defaultPullInterval,
	}
	return r
}

func (r *Registry) Alarms() (map[string]models.Alarm, error) {
	var resp RespAlarm
	alarms := make(map[string]models.Alarm)
	url := fmt.Sprintf("%s/api/v1/alarm/resource?ns=%s&type=alarm", r.Addr, root)
	response, err := requests.Get(url)
	if err != nil {
		return alarms, err
	}

	if response.Status == 200 {
		err = json.Unmarshal(response.Body, &resp)
		if err != nil {
			return alarms, err
		}
		for _, a := range resp.Data {
			alarms[a.Version] = a
		}
		return alarms, nil
	}

	return alarms, fmt.Errorf("get alarms failed: code %d", response.Status)
}

func (r *Registry) AlarmServers() ([]string, error) {
	var resp RespMachine
	var servers []string
	url := fmt.Sprintf("%s/api/v1/alarm/resource?ns=%s&type=machine", r.Addr, r.AlarmNS)
	response, err := requests.Get(url)
	if err != nil {
		return servers, err
	}

	if response.Status == 200 {
		err = json.Unmarshal(response.Body, &resp)
		if err != nil {
			return servers, err
		}
		for _, a := range resp.Data {
			ip := IntranetIP(a.IP)
			servers = append(servers, ip)
		}
		return servers, nil
	}

	return servers, fmt.Errorf("get alarm servers failed: code %d", response.Status)
}

func IntranetIP(ipStr string) string {
	ips := strings.Split(ipStr, ",")
	if len(ips) == 1 {
		return ipStr
	}
	for _, ip := range ips {
		matched, _ := regexp.MatchString(REGIntrannetIP, ip)
		if matched {
			return ip
		}
	}
	return ips[0]
}
