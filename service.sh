#!/bin/bash

chmod a+x /usr/local/alarm-adapter/alarm-adapter
/usr/local/alarm-adapter/alarm-adapter stop

sleep 3

nohup /usr/local/alarm-adapter/alarm-adapter start -c /usr/local/alarm-adapter/conf/alarm-adapter.conf > /dev/null 2>&1 &
