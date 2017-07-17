package snmp

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lodastack/alarm-adapter/report"
	"github.com/lodastack/models"
	"github.com/lodastack/sdk-go"

	"github.com/alouca/gosnmp"
	"github.com/lodastack/log"
)

const (
	// SNMP OID CONST
	OID_INF_INFO        = ".1.3.6.1.2.1.2.2.1.2"
	OID_INF_INDEX       = "1.3.6.1.2.1.2.2.1.1"
	OID_INF_TRAFFIC_IN  = "1.3.6.1.2.1.31.1.1.1.6"
	OID_INF_TRAFFIC_OUT = "1.3.6.1.2.1.31.1.1.1.10"
	OID_INF_STATUS      = ".1.3.6.1.2.1.2.2.1.8"
	OID_SYS_HOSTNAME    = ".1.3.6.1.2.1.1.5.0"

	SNMP_TIMEOUT int64 = 10 //unit: sec

	TYPE_IN  = "in"
	TYPE_OUT = "out"
)

var (
	historymap  map[string]int64
	lasttimemap map[string]int64
	mu          sync.RWMutex
)

type snmpServer struct {
	ip        string
	community string
}

func traffic(ns string, ip string, hostname string, community []string) {
	var s *gosnmp.GoSNMP
	var err error
	for _, communityString := range community {
		s, err = gosnmp.NewGoSNMP(ip, communityString, gosnmp.Version2c, SNMP_TIMEOUT)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Errorf("connect server %s failed: %s", ip, err.Error())
		return
	}
	s.SetTimeout(50)
	NetworkInfs := FetchInfIndex(s)
	Points := FetchTraffic(ns, s, NetworkInfs, ip, hostname)
	go Send(ns, Points)
	go report.Report(hostname, ip, ns)
}

type NetworkInf struct {
	oid       string
	valuetype string
	name      string
	index     int
}

func FetchInfIndex(s *gosnmp.GoSNMP) (NetworkInfs []NetworkInf) {
	resp, err := s.Walk(OID_INF_INDEX)
	if err == nil {
		for _, netinterface := range resp {
			var one NetworkInf
			one.oid = netinterface.Name
			one.valuetype = string(netinterface.Type)
			//avoid panic risk
			var ok bool
			one.index, ok = netinterface.Value.(int)
			if !ok {
				log.Errorf("interface index type is not int : %v", netinterface.Value)
			}
			NetworkInfs = append(NetworkInfs, one)
		}
	} else {
		log.Errorf("Get net interface failed: %s", err.Error())
	}
	return
}

type NetworkTraffic struct {
	oid       string
	valuetype string
	invalue   int64
	outvalue  int64
	status    int
}

func FetchTraffic(ns string, s *gosnmp.GoSNMP, NetworkInfs []NetworkInf, ip string, hostname string) (points []models.Metric) {
	// ifHCInOctets

	for _, netinterface := range NetworkInfs {
		var one NetworkTraffic
		var ok bool

		index := fmt.Sprintf("%d", netinterface.index)
		info_resp, info_err := s.Get(fmt.Sprintf("%s.%s", OID_INF_INFO, index))
		if info_err != nil {
			log.Errorf("get interface info failed: %s", info_err)
			continue
		}

		if len(info_resp.Variables) > 0 {
			one.oid, ok = info_resp.Variables[0].Value.(string)
			netinterface.name = one.oid
			if !ok {
				log.Errorf("interface name type is not string : %v", info_resp.Variables[0].Value)
			}
		}

		in_resp, in_err := s.Get(fmt.Sprintf("%s.%s", OID_INF_TRAFFIC_IN, index))
		out_resp, out_err := s.Get(fmt.Sprintf("%s.%s", OID_INF_TRAFFIC_OUT, index))
		status_resp, status_err := s.Get(fmt.Sprintf("%s.%s", OID_INF_STATUS, index))
		if in_err != nil || out_err != nil || status_err != nil {
			continue
		}

		//avoid panic risk
		if len(in_resp.Variables) > 0 {
			one.invalue, ok = in_resp.Variables[0].Value.(int64)
			if !ok {
				log.Errorf("in traffic type is not int64 : %v", in_resp.Variables[0].Value)
			}
		}

		if len(out_resp.Variables) > 0 {
			one.outvalue, ok = out_resp.Variables[0].Value.(int64)
			if !ok {
				log.Errorf("out traffic type is not int64 : %v", out_resp.Variables[0].Value)
			}
		}

		if len(status_resp.Variables) > 0 {
			one.status, ok = status_resp.Variables[0].Value.(int)
			if !ok {
				log.Errorf("inf status is not int64 : %v", status_resp.Variables[0].Value)
			}
		}
		log.Infof("NetworkInfs index = %d", netinterface.index)
		points = append(points, MakePoint(ns, one, netinterface, ip, hostname)...)
	}
	return
}

func MakePoint(ns string, t NetworkTraffic, i NetworkInf, ip string, hostname string) (pair []models.Metric) {
	mu.Lock()
	defer mu.Unlock()
	timekey := fmt.Sprint("%s%s%s", ns, ip, i.name)
	inkey := fmt.Sprint("%s%s%s%s", ns, ip, i.name, TYPE_IN)
	outkey := fmt.Sprint("%s%s%s%s", ns, ip, i.name, TYPE_OUT)
	if _, ok := historymap[inkey]; !ok {
		historymap[inkey] = t.invalue
		historymap[outkey] = t.outvalue
		lasttimemap[timekey] = time.Now().Unix()
		log.Infof("history data init [%s %s]", ip, i.name)
		return pair
	}

	if _, ok := lasttimemap[timekey]; !ok {
		lasttimemap[timekey] = time.Now().Unix()
		log.Infof("history time data init [%s %s]", ns, ip)
	}

	if time.Now().Unix()-lasttimemap[timekey] > (int64)(DefaultInterval*1.5) {
		log.Errorf("last time expired [%d]", lasttimemap[timekey])
		lasttimemap[timekey] = time.Now().Unix()
		delete(historymap, outkey)
		delete(historymap, inkey)
		return pair
	}

	point_in := models.Metric{
		Name:      "net.traffic." + hostname,
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_IN,
		},
		Value: ((t.invalue - historymap[inkey]) / (int64)(DefaultInterval)) * 8,
	}

	point_in_normal := models.Metric{
		Name:      "net.traffic",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_IN,
		},
		Value: ((t.invalue - historymap[inkey]) / (int64)(DefaultInterval)) * 8,
	}

	point_out := models.Metric{
		Name:      "net.traffic." + hostname,
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_OUT,
		},
		Value: ((t.outvalue - historymap[outkey]) / (int64)(DefaultInterval)) * 8,
	}

	point_out_normal := models.Metric{
		Name:      "net.traffic",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_OUT,
		},
		Value: ((t.outvalue - historymap[outkey]) / (int64)(DefaultInterval)) * 8,
	}

	point_status := models.Metric{
		Name:      "net.infStatus",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"host": hostname,
			"if":   i.name,
		},
		Value: t.status,
	}

	point_status_host := models.Metric{
		Name:      "net.infStatus." + hostname,
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"host": hostname,
			"if":   i.name,
		},
		Value: t.status,
	}

	historymap[outkey] = t.outvalue
	historymap[inkey] = t.invalue
	lasttimemap[timekey] = time.Now().Unix()

	pair = append(pair, point_in)
	pair = append(pair, point_out)
	pair = append(pair, point_in_normal)
	pair = append(pair, point_out_normal)
	pair = append(pair, point_status)
	pair = append(pair, point_status_host)

	log.Infof("make points: [ip:%s, if:%s, in-traffic:%d, out-traffic:%d]", ip, i.name, t.invalue, t.outvalue)
	return pair
}

func Send(ns string, ms []models.Metric) error {
	data, err := json.Marshal(ms)
	if err != nil {
		return err
	}
	return sdk.Post(ns, data)
}
