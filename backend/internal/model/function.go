package model

import "time"

type RuntimeName string

const (
	RuntimeGo     RuntimeName = "go"
	RuntimeNodeJS RuntimeName = "nodejs"
)

func (r RuntimeName) Valid() bool {
	return r == RuntimeGo || r == RuntimeNodeJS
}

type FunctionStatus string

const (
	StatusDraft    FunctionStatus = "DRAFT"
	StatusBuilding FunctionStatus = "BUILDING"
	StatusReady    FunctionStatus = "READY"
	StatusFailed   FunctionStatus = "FAILED"
)

func (s FunctionStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusBuilding, StatusReady, StatusFailed:
		return true
	default:
		return false
	}
}

type Function struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Runtime        RuntimeName       `json:"runtime"`
	Status         FunctionStatus    `json:"status"`
	Description    string            `json:"description"`
	TimeoutSec     int               `json:"timeout_sec"`
	MemoryMB       int               `json:"memory_mb"`
	CPUNano        int64             `json:"cpu_nano"`
	MaxConcurrency int               `json:"max_concurrency"`
	Env            map[string]string `json:"env"`
	CurrentVersion int               `json:"current_version"`
	Endpoint       string            `json:"endpoint"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type FunctionVersion struct {
	ID           string         `json:"id"`
	FunctionID   string         `json:"function_id"`
	Version      int            `json:"version"`
	Status       FunctionStatus `json:"status"`
	Code         string         `json:"code"`
	ArtifactPath string         `json:"artifact_path"`
	BuildLog     string         `json:"build_log"`
	CreatedAt    time.Time      `json:"created_at"`
}

type TriggerKind string

const (
	TriggerHTTP TriggerKind = "http"
	TriggerCron TriggerKind = "cron"
)

func (k TriggerKind) Valid() bool {
	return k == TriggerHTTP || k == TriggerCron
}

type Trigger struct {
	ID         string      `json:"id"`
	FunctionID string      `json:"function_id"`
	Kind       TriggerKind `json:"kind"`
	CronExpr   string      `json:"cron_expr,omitempty"`
	Enabled    bool        `json:"enabled"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type Invocation struct {
	ID          string      `json:"id"`
	FunctionID  string      `json:"function_id"`
	Name        string      `json:"function_name,omitempty"`
	Version     int         `json:"version"`
	TriggerKind TriggerKind `json:"trigger_kind"`
	StatusCode  int         `json:"status_code"`
	Success     bool        `json:"success"`
	ColdStart   bool        `json:"cold_start"`
	WakeupMS    int64       `json:"wakeup_ms"`
	ExecMS      int64       `json:"exec_ms"`
	E2EMS       int64       `json:"e2e_ms"`
	Error       string      `json:"error,omitempty"`
	Logs        string      `json:"logs,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type FunctionMetrics struct {
	FunctionName   string  `json:"function_name"`
	Invocations    int64   `json:"invocations"`
	Successes      int64   `json:"successes"`
	Failures       int64   `json:"failures"`
	ColdStarts     int64   `json:"cold_starts"`
	AvgExecMS      float64 `json:"avg_exec_ms"`
	P95ExecMS      float64 `json:"p95_exec_ms"`
	P99ExecMS      float64 `json:"p99_exec_ms"`
	AvgWakeupMS    float64 `json:"avg_wakeup_ms"`
	LastInvokeAt   string  `json:"last_invoke_at,omitempty"`
	PoolHits       int64   `json:"pool_hits"`
	PoolMisses     int64   `json:"pool_misses"`
	ActiveSlots    int     `json:"active_slots"`
	IdleSlots      int     `json:"idle_slots"`
}

type CreateFunctionInput struct {
	Name           string            `json:"name"`
	Runtime        RuntimeName       `json:"runtime"`
	Description    string            `json:"description"`
	Code           string            `json:"code"`
	TimeoutSec     int               `json:"timeout_sec"`
	MemoryMB       int               `json:"memory_mb"`
	MaxConcurrency int               `json:"max_concurrency"`
	Env            map[string]string `json:"env"`
}

type UpdateFunctionInput struct {
	Description    *string           `json:"description"`
	Code           *string           `json:"code"`
	TimeoutSec     *int              `json:"timeout_sec"`
	MemoryMB       *int              `json:"memory_mb"`
	MaxConcurrency *int              `json:"max_concurrency"`
	Env            map[string]string `json:"env"`
}

type DeployInput struct {
	Code string `json:"code"`
}

type RollbackInput struct {
	Version int `json:"version"`
}

type UpsertTriggersInput struct {
	Triggers []Trigger `json:"triggers"`
}
