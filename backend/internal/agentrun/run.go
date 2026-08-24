package agentrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

type Request struct {
	RequestID string            `json:"request_id"`
	Command   []string          `json:"command"`
	WorkDir   string            `json:"workdir"`
	TimeoutMS int               `json:"timeout_ms"`
	Env       map[string]string `json:"env"`
	EventJSON string            `json:"event_json"`
}

type Response struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	DurationMS int64             `json:"duration_ms"`
	Logs       string            `json:"logs"`
	Error      string            `json:"error,omitempty"`
}

type userOut struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func Run(ctx context.Context, req Request) Response {
	if len(req.Command) == 0 {
		return Response{Status: 500, Error: "empty command"}
	}
	timeout := time.Duration(req.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}
	cmd.Stdin = strings.NewReader(req.EventJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = SafeEnv(req.Env)
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()
	if dur <= 0 {
		dur = 1
	}
	logs := strings.TrimSpace(stderr.String())
	if ctx.Err() == context.DeadlineExceeded {
		return Response{Status: 504, DurationMS: dur, Logs: logs, Error: "execution timeout"}
	}
	if err != nil && stdout.Len() == 0 {
		return Response{Status: 500, DurationMS: dur, Logs: logs, Error: err.Error()}
	}
	var out userOut
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &out); err != nil {
		return Response{
			Status: 200, Headers: map[string]string{"content-type": "text/plain"},
			Body: stdout.String(), DurationMS: dur, Logs: logs,
		}
	}
	if out.Status == 0 {
		out.Status = 200
	}
	if out.Headers == nil {
		out.Headers = map[string]string{}
	}
	return Response{Status: out.Status, Headers: out.Headers, Body: out.Body, DurationMS: dur, Logs: logs}
}

func SafeEnv(extra map[string]string) []string {
	base := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/tmp",
		"TMPDIR=/tmp",
		"TZ=Asia/Shanghai",
	}
	for k, v := range extra {
		if !ValidEnvKey(k) {
			continue
		}
		base = append(base, k+"="+v)
	}
	return base
}

func ValidEnvKey(k string) bool {
	if k == "" || len(k) > 64 {
		return false
	}
	for i, r := range k {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}
