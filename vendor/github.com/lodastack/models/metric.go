package models

type Metric struct {
	Name      string            `json:"name"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
	Value     interface{}       `json:"value"`
	Offset    int64             `json:"offset,omitempty"`
}
