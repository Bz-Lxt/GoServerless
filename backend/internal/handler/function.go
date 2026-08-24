package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/gogo/goserverless/internal/logstream"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/service"
	"github.com/gogo/goserverless/internal/timeutil"
)

type API struct {
	fn  *service.Functions
	hub *logstream.Hub
}

func NewAPI(fn *service.Functions, hub *logstream.Hub) *API {
	return &API{fn: fn, hub: hub}
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	items, err := a.fn.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, items)
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	var in model.CreateFunctionInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	fn, err := a.fn.Create(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeCreated(w, fn)
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	fn, code, err := a.fn.Get(r.Context(), name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"function": fn, "code": code})
}

func (a *API) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var in model.UpdateFunctionInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	fn, err := a.fn.Update(r.Context(), name, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, fn)
}

func (a *API) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := a.fn.Delete(r.Context(), name); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]bool{"ok": true})
}

func (a *API) Deploy(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var in model.DeployInput
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErr(w, model.Invalid("malformed json"))
			return
		}
	}
	fn, err := a.fn.Deploy(r.Context(), name, in.Code)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, fn)
}

func (a *API) Rollback(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var in model.RollbackInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	fn, err := a.fn.Rollback(r.Context(), name, in.Version)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, fn)
}

func (a *API) Versions(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	items, err := a.fn.Versions(r.Context(), name)
	if err != nil {
		writeErr(w, err)
		return
	}
	type row struct {
		Version   int    `json:"version"`
		Status    string `json:"status"`
		BuildLog  string `json:"build_log"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]row, 0, len(items))
	for _, v := range items {
		out = append(out, row{
			Version:   v.Version,
			Status:    string(v.Status),
			BuildLog:  v.BuildLog,
			CreatedAt: timeutil.Format(v.CreatedAt),
		})
	}
	writeOK(w, out)
}

func (a *API) Triggers(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	items, err := a.fn.Triggers(r.Context(), name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, items)
}

func (a *API) PutTriggers(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var in model.UpsertTriggersInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	items, err := a.fn.ReplaceTriggers(r.Context(), name, in.Triggers)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, items)
}

func (a *API) Metrics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	m, err := a.fn.Metrics(r.Context(), name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, m)
}

func (a *API) Invocations(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := a.fn.Invocations(r.Context(), name, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	type row struct {
		ID         string `json:"id"`
		Version    int    `json:"version"`
		Trigger    string `json:"trigger_kind"`
		StatusCode int    `json:"status_code"`
		Success    bool   `json:"success"`
		ColdStart  bool   `json:"cold_start"`
		WakeupMS   int64  `json:"wakeup_ms"`
		ExecMS     int64  `json:"exec_ms"`
		E2EMS      int64  `json:"e2e_ms"`
		Error      string `json:"error"`
		Logs       string `json:"logs"`
		CreatedAt  string `json:"created_at"`
	}
	out := make([]row, 0, len(items))
	for _, it := range items {
		out = append(out, row{
			ID: it.ID, Version: it.Version, Trigger: string(it.TriggerKind),
			StatusCode: it.StatusCode, Success: it.Success, ColdStart: it.ColdStart,
			WakeupMS: it.WakeupMS, ExecMS: it.ExecMS, E2EMS: it.E2EMS,
			Error: it.Error, Logs: it.Logs, CreatedAt: timeutil.Format(it.CreatedAt),
		})
	}
	writeOK(w, out)
}

func (a *API) Overview(w http.ResponseWriter, r *http.Request) {
	m, err := a.fn.Overview(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, m)
}

func (a *API) Templates(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{
		"go":     a.fn.Template(model.RuntimeGo),
		"nodejs": a.fn.Template(model.RuntimeNodeJS),
	})
}

func (a *API) Logs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, _, err := a.fn.Get(r.Context(), name); err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, model.Unavailable("streaming unsupported"))
		return
	}
	ch, cancel := a.hub.Subscribe(name)
	defer cancel()
	fmt.Fprintf(w, "event: hello\ndata: {\"function\":\"%s\"}\n\n", name)
	flusher.Flush()
	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case raw, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", raw)
			flusher.Flush()
		}
	}
}
