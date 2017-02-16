package requests

import (
	"net/http"
)

func Get(url string) (*Resp, error) {
	var rs *Resp = new(Resp)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	rs.Status = resp.StatusCode

	if bytes, err := getBytes(resp); err != nil {
		return nil, err
	} else {
		rs.Body = bytes
		return rs, nil
	}
}
