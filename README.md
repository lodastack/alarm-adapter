
# Alarm Adapter [![CircleCI](https://circleci.com/gh/lodastack/alarm-adapter.svg?style=svg&circle-token=67ea071b179f21ae2592ec4759eaa0777eb42472)](https://circleci.com/gh/lodastack/alarm-adapter)

The main func is write user alarms into kapacitor and update if user change the config. For monitoring API status, Ping status and support switch SNMP collect.

## Build

    make build
    
## Start alarm-adapter
    
    alarm-adapter start -f ${path_to_config_file}

## Stop agent

    alarm-adapter stop

## Configuration

```
[main]
	#registry service address
	registryAddr  = "http://registry.test.com"
	
[alarm]
	enable        = true
	#kapacitor NS
	NS            = "kapacitor.alarm.monitor.loda"
	eventAddr     = "http://event.test.com"

[ping]
	enable        = false
	ipList        = ["10.50.","10.90."]

[api]
	enable        = false
	#All api will be monitored if global is true, and the NS will be ignored
	global        = false
	NS            = ["api.loda"]

[snmp]
	enable        = false
	NS            = ["switch.loda"]
	ipList        = ["10.50.","10.90."]
	community     = ["test","test2"]


[log]
	logdir        = "/tmp/alarm-adapter/log"
	loglevel      = "INFO"
	logrotatenum  = 5
	logrotatesize = 1887436800


```
