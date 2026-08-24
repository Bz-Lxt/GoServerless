package invoker

import "time"

type HTTPEvent struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type AgentRequest struct {
	RequestID string            `json:"request_id"`
	Command   []string          `json:"command"`
	WorkDir   string            `json:"workdir"`
	TimeoutMS int               `json:"timeout_ms"`
	Env       map[string]string `json:"env"`
	EventJSON string            `json:"event_json"`
}

type AgentResponse struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	DurationMS int64             `json:"duration_ms"`
	Logs       string            `json:"logs"`
	Error      string            `json:"error,omitempty"`
}

type Result struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	ColdStart  bool
	Wakeup     time.Duration
	Exec       time.Duration
	E2E        time.Duration
	Logs       string
	Error      string
	Version    int
}
