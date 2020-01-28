## golang SDK

example code:

```

import "github.com/lodastack/sdk-go"

type Metric struct {
	Name      string            `json:"name"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
	Value     interface{}       `json:"value"`
	Offset    int64             `json:"offset,omitempty"`
}

func Send(ns string, from string, loss int) error {
	m := Metric{
		Name:      "ping.loss",
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"from": from,
		},
		Value: loss,
	}
	data, err := json.Marshal([]Metric{m})
	if err != nil {
		return err
	}
	return sdk.Post(ns, data)
}

```
