package sdk

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
)

// monitor agent socket file
const MonitorAgentSocket = "/var/run/monitor-agent.sock"

// all your client.Get or client.Post calls has to be a valid url (http://xxxx.xxx/path not unix://...),
// the domain name doesn't matter since it won't be used in connecting.
const UnixSocketURL = "http://loda:1234/post?ns=%s"

var client *http.Client

func init() {
	tr := &http.Transport{
		Dial: unixsocketDial,
	}
	client = &http.Client{Transport: tr}
}

func unixsocketDial(proto, addr string) (conn net.Conn, err error) {
	return net.Dial("unix", MonitorAgentSocket)
}

func Post(ns string, data []byte) error {
	body := bytes.NewBuffer(data)
	URL := fmt.Sprintf(UnixSocketURL, ns)
	req, err := http.NewRequest("POST", URL, body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("Post data failed StatusCode:%d", resp.StatusCode)
}
