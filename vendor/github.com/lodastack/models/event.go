package models

// Event type struct
type Event struct {
	ID        string   `json:"id"`
	NS        []string `json:"ns"`
	Machine   []string `json:"machine"`
	Message   string   `json:"message"`
	Link      string   `json:"link"`
	Author    string   `json:"author"`
	Timestamp int64    `json:"timestamp"`
	Type      string   `json:"type"`
	App       string   `json:"app"`
	Token     string   `json:"token"`
}

// Events
type Events []Event
