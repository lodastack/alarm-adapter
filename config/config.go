package config

import (
	"sync"

	"github.com/BurntSushi/toml"
)

const (
	//APP NAME
	AppName = "Alarm Adapter"
	//Usage
	Usage = "Alarm Adapter Usage"
	//Vresion Num
	Version = "0.0.1"
	//Author Nmae
	Author = "Devlopers LodaStack"
	//Email Address
	Email = "devlopers@lodastack.com"
)

const (
	//PID FILE
	PID = "/var/run/alarm-adapter.pid"
)

var (
	mux = new(sync.RWMutex)
	C   = new(Config)
)

type Config struct {
	Main  MainConfig  `toml:"main"`
	Alarm AlarmConfig `toml:"alarm"`
	Ping  PingConfig  `toml:"ping"`
	API   APIConfig   `toml:"api"`
	SNMP  SNMPConfig  `toml"snmp"`
	Log   LogConfig   `toml:"log"`
}

type MainConfig struct {
	RegistryAddr string `toml:"registryAddr"`
}

type AlarmConfig struct {
	Enable    bool   `toml:"enable"`
	NS        string `toml:"NS"`
	EventAddr string `toml:"eventAddr"`
}

type PingConfig struct {
	Enable bool     `toml:"enable"`
	IpList []string `toml:"ipList"`
}

type APIConfig struct {
	Enable bool     `toml:"enable"`
	Global bool     `toml:"global"`
	NS     []string `toml:"NS"`
}

type SNMPConfig struct {
	Enable    bool     `toml:"enable"`
	NS        []string `toml:"NS"`
	IpList    []string `toml:"ipList"`
	Community string   `toml:"community"`
}

type LogConfig struct {
	Dir           string `toml:"logdir"`
	Level         string `toml:"loglevel"`
	Logrotatenum  int    `toml:"logrotatenum"`
	Logrotatesize uint64 `toml:"logrotatesize"`
}

func ParseConfig(path string) error {
	mux.Lock()
	defer mux.Unlock()

	if _, err := toml.DecodeFile(path, &C); err != nil {
		return err
	}
	return nil
}

func GetConfig() *Config {
	mux.RLock()
	defer mux.RUnlock()
	return C
}
