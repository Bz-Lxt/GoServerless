package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gogo/goserverless/internal/model"
	"github.com/robfig/cron/v3"
)

var (
	fnNameRe  = regexp.MustCompile(`^[a-z][a-z0-9-]{2,39}$`)
	envKeyRe  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
	cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

const (
	MaxCodeBytes     = 256 << 10
	MaxEnvPairs      = 32
	MaxEnvValueBytes = 4 << 10
	MinTimeoutSec    = 1
	MaxTimeoutSec    = 300
	MinNameLen       = 3
	MaxNameLen       = 40
)

var allowedMemory = map[int]struct{}{64: {}, 128: {}, 256: {}, 512: {}}

func FunctionName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Invalid("function name is required")
	}
	if utf8.RuneCountInString(name) < MinNameLen || utf8.RuneCountInString(name) > MaxNameLen {
		return model.Invalid("function name must be 3-40 characters")
	}
	if !fnNameRe.MatchString(name) {
		return model.Invalid("function name must match ^[a-z][a-z0-9-]{2,39}$")
	}
	return nil
}

func Runtime(r model.RuntimeName) error {
	if !r.Valid() {
		return model.Invalid("runtime must be go or nodejs")
	}
	return nil
}

func Code(code string) error {
	if strings.TrimSpace(code) == "" {
		return model.Invalid("code is required")
	}
	if len(code) > MaxCodeBytes {
		return model.Invalid(fmt.Sprintf("code exceeds %d bytes", MaxCodeBytes))
	}
	return nil
}

func Timeout(sec int) error {
	if sec < MinTimeoutSec || sec > MaxTimeoutSec {
		return model.Invalid("timeout_sec must be 1-300")
	}
	return nil
}

func MemoryMB(mb int) error {
	if _, ok := allowedMemory[mb]; !ok {
		return model.Invalid("memory_mb must be one of 64, 128, 256, 512")
	}
	return nil
}

func Concurrency(n int) error {
	if n < 1 || n > 50 {
		return model.Invalid("max_concurrency must be 1-50")
	}
	return nil
}

func Env(env map[string]string) error {
	if env == nil {
		return nil
	}
	if len(env) > MaxEnvPairs {
		return model.Invalid("env exceeds 32 pairs")
	}
	for k, v := range env {
		if !envKeyRe.MatchString(k) {
			return model.Invalid(fmt.Sprintf("invalid env key %q", k))
		}
		if len(v) > MaxEnvValueBytes {
			return model.Invalid(fmt.Sprintf("env value for %q exceeds 4KiB", k))
		}
		if strings.ContainsAny(k, " \t\n$`'\"\\|&;<>") {
			return model.Invalid(fmt.Sprintf("env key %q contains unsafe characters", k))
		}
	}
	return nil
}

func CronExpr(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return model.Invalid("cron_expr is required for cron trigger")
	}
	if _, err := cronParser.Parse(expr); err != nil {
		return model.Invalid("invalid cron_expr: " + err.Error())
	}
	return nil
}

func Triggers(items []model.Trigger) error {
	if len(items) == 0 {
		return model.Invalid("at least one trigger is required")
	}
	if len(items) > 8 {
		return model.Invalid("at most 8 triggers")
	}
	hasHTTP := false
	for i, t := range items {
		if !t.Kind.Valid() {
			return model.Invalid(fmt.Sprintf("triggers[%d].kind must be http or cron", i))
		}
		if t.Kind == model.TriggerHTTP {
			hasHTTP = true
			if strings.TrimSpace(t.CronExpr) != "" {
				return model.Invalid("http trigger must not carry cron_expr")
			}
		}
		if t.Kind == model.TriggerCron {
			if err := CronExpr(t.CronExpr); err != nil {
				return err
			}
		}
	}
	if !hasHTTP {
		return model.Invalid("http trigger is mandatory")
	}
	return nil
}

func CreateInput(in model.CreateFunctionInput) error {
	if err := FunctionName(in.Name); err != nil {
		return err
	}
	if err := Runtime(in.Runtime); err != nil {
		return err
	}
	if err := Code(in.Code); err != nil {
		return err
	}
	if in.TimeoutSec == 0 {
		in.TimeoutSec = 30
	}
	if in.MemoryMB == 0 {
		in.MemoryMB = 128
	}
	if in.MaxConcurrency == 0 {
		in.MaxConcurrency = 10
	}
	if err := Timeout(in.TimeoutSec); err != nil {
		return err
	}
	if err := MemoryMB(in.MemoryMB); err != nil {
		return err
	}
	if err := Concurrency(in.MaxConcurrency); err != nil {
		return err
	}
	if err := Env(in.Env); err != nil {
		return err
	}
	if utf8.RuneCountInString(in.Description) > 200 {
		return model.Invalid("description exceeds 200 characters")
	}
	return nil
}
