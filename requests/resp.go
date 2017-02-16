package requests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Resp struct {
	Status int
	Body   []byte
}

func (r *Resp) Map() (map[string]interface{}, error) {
	var rs = make(map[string]interface{})
	err := json.Unmarshal(r.Body, &rs)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *Resp) Obj(obj interface{}) error {
	err := json.Unmarshal(r.Body, obj)
	if err != nil {
		return err
	}
	return nil
}

func (r *Resp) Slice() ([]interface{}, error) {
	var rs []interface{}
	err := json.Unmarshal(r.Body, &rs)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func getBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	return body, nil
}
