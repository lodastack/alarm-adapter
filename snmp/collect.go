package snmp

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lodastack/models"
	"github.com/lodastack/sdk-go"

	"github.com/alouca/gosnmp"
	"github.com/lodastack/log"
)

const (
	// SNMP OID CONST
	OID_INF_INFO        = ".1.3.6.1.2.1.2.2.1.2"
	OID_INF_TRAFFIC_IN  = "1.3.6.1.2.1.31.1.1.1.6"
	OID_INF_TRAFFIC_OUT = "1.3.6.1.2.1.31.1.1.1.10"
	OID_SYS_HOSTNAME    = ".1.3.6.1.2.1.1.5.0"

	SNMP_TIMEOUT int64 = 3 //unit: sec

	TYPE_IN  = "in"
	TYPE_OUT = "out"
)

var (
	historymap map[string]int64
	mu         sync.RWMutex
)

type snmpServer struct {
	ip        string
	community string
}

func traffic(ns string, ip string, hostname string, community string) {
	s, err := gosnmp.NewGoSNMP(ip, community, gosnmp.Version2c, SNMP_TIMEOUT)
	if err != nil {
		log.Errorf("connect server %s failed: %s", ip, err.Error())
		return
	}
	NetworkInfs := FetchInfs(s)
	Points := FetchTraffic(ns, s, NetworkInfs, ip, hostname)
	go Send(ns, Points)
}

type NetworkInf struct {
	oid       string
	valuetype string
	name      string
}

func FetchInfs(s *gosnmp.GoSNMP) (NetworkInfs []NetworkInf) {
	resp, err := s.Walk(OID_INF_INFO)
	if err == nil {
		for _, netinterface := range resp {
			var one NetworkInf
			one.oid = netinterface.Name
			one.valuetype = string(netinterface.Type)
			//avoid panic risk
			var ok bool
			one.name, ok = netinterface.Value.(string)
			if !ok {
				log.Errorf("interface name type is not string : %v", netinterface.Value)
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
}

func FetchTraffic(ns string, s *gosnmp.GoSNMP, NetworkInfs []NetworkInf, ip string, hostname string) (points []models.Metric) {
	// ifHCInOctets
	in_resp, in_err := s.Walk(OID_INF_TRAFFIC_IN)
	out_resp, out_err := s.Walk(OID_INF_TRAFFIC_OUT)
	if in_err == nil && out_err == nil && len(in_resp) == len(NetworkInfs) {
		for i, netinterface := range in_resp {
			var one NetworkTraffic
			one.oid = netinterface.Name
			one.valuetype = string(netinterface.Type)
			//avoid panic risk
			var ok bool
			one.invalue, ok = netinterface.Value.(int64)
			if !ok {
				log.Errorf("in traffic type is not int64 : %v", netinterface.Value)
			}
			one.outvalue, ok = out_resp[i].Value.(int64)
			if !ok {
				log.Errorf("out traffic type is not int64 : %v", out_resp[i].Value)
			}
			log.Infof("NetworkInfs len = %d, index = %d", len(NetworkInfs), i)
			points = append(points, MakePoint(ns, one, NetworkInfs[i], ip, hostname)...)
		}
	} else {
		log.Errorf("Get net interface in traffic failed: IN: %s Out: %s len(in_resp): %d len(NetworkInfs): %d", in_err, out_err, len(in_resp), len(NetworkInfs))
	}
	return
}

func MakePoint(ns string, t NetworkTraffic, i NetworkInf, ip string, hostname string) (pair []models.Metric) {
	mu.Lock()
	defer mu.Unlock()
	inkey := fmt.Sprint("%s%s%s%s", ns, ip, i.name, TYPE_IN)
	outkey := fmt.Sprint("%s%s%s%s", ns, ip, i.name, TYPE_OUT)
	if _, ok := historymap[inkey]; !ok {
		historymap[inkey] = t.invalue
		historymap[outkey] = t.outvalue
		log.Infof("history data init [%s %s]", ip, i.name)
		return pair
	}

	if _, ok := historymap[outkey]; !ok {
		historymap[inkey] = t.invalue
		historymap[outkey] = t.outvalue
		log.Infof("history data init [%s %s]", ip, i.name)
		return pair
	}

	point_in := models.Metric{
		Name:      "net.traffic",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_IN,
		},
		Value: (t.invalue - historymap[inkey]) / (int64)(DefaultInterval),
	}

	point_out := models.Metric{
		Name:      "net.traffic",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": Hostname,
			"host": hostname,
			"if":   i.name,
			"type": TYPE_OUT,
		},
		Value: (t.outvalue - historymap[outkey]) / (int64)(DefaultInterval),
	}

	historymap[outkey] = t.outvalue
	historymap[inkey] = t.invalue

	pair = append(pair, point_in)
	pair = append(pair, point_out)

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
