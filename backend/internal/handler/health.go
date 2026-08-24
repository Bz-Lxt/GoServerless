package handler

import (
	"net/http"

	"github.com/gogo/goserverless/internal/timeutil"
)

func Health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{
		"status": "ok",
		"time":   timeutil.Format(timeutil.Now()),
		"tz":     "Asia/Shanghai",
	})
}
