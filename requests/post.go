package requests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func PostBytes(url string, data []byte) (*Resp, error) {
	return doPostPain(url, bytes.NewReader(data))
}

func Post(url string, obj interface{}) (*Resp, error) {
	data, err := getReader(obj)
	if err != nil {
		return nil, err
	}
	return doPost(url, data)
}

func doPost(url string, data io.Reader) (*Resp, error) {
	resp, err := http.Post(url, "application/json", data)
	return post(resp, err)
}

func post(resp *http.Response, err error) (*Resp, error) {
	if err != nil {
		return nil, err
	}

	response := new(Resp)
	response.Status = resp.StatusCode
	if body, err := getBytes(resp); err != nil {
		return nil, err
	} else {
		response.Body = body
		return response, nil
	}
}

func doPostPain(url string, data io.Reader) (*Resp, error) {
	resp, err := http.Post(url, "text/plain", data)
	return post(resp, err)
}

func getReader(obj interface{}) (io.Reader, error) {
	if bs, err := json.Marshal(&obj); err != nil {
		return nil, err
	} else {
		rs := bytes.NewReader(bs)
		return rs, nil
	}
}
