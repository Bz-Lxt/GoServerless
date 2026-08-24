package gateway

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/timeutil"
)

type Recorder interface {
	Record(fn *model.Function, kind model.TriggerKind, res *invoker.Result)
}

type Resolver interface {
	ReadyFunction(name string) (*model.Function, error)
}

type Gateway struct {
	inv      *invoker.Invoker
	rec      Recorder
	maxBody  int64
	resolver func(name string) (*model.Function, error)
}

func New(inv *invoker.Invoker, rec Recorder, maxBody int64) *Gateway {
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	return &Gateway{inv: inv, rec: rec, maxBody: maxBody}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	routeStart := time.Now()
	name := chi.URLParam(r, "function_name")
	if name == "" {
		name = chi.URLParam(r, "name")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		http.Error(w, `{"code":"INVALID_INPUT","message":"function_name required"}`, http.StatusBadRequest)
		return
	}
	if r.ContentLength > g.maxBody {
		http.Error(w, `{"code":"TOO_LARGE","message":"body exceeds 1MB"}`, http.StatusRequestEntityTooLarge)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, g.maxBody+1))
	if err != nil {
		http.Error(w, `{"code":"INVALID_INPUT","message":"read body"}`, http.StatusBadRequest)
		return
	}
	if int64(len(body)) > g.maxBody {
		http.Error(w, `{"code":"TOO_LARGE","message":"body exceeds 1MB"}`, http.StatusRequestEntityTooLarge)
		return
	}

	headers := map[string]string{}
	for k, vs := range r.Header {
		if len(vs) == 0 {
			continue
		}
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "cookie" {
			continue
		}
		headers[k] = vs[0]
	}
	query := map[string]string{}
	for k, vs := range r.URL.Query() {
		if len(vs) > 0 {
			query[k] = vs[0]
		}
	}
	path := StripInvokePrefix(r.URL.Path, name)

	routeMS := timeutil.DurationMS(time.Since(routeStart))
	ev := invoker.HTTPEvent{
		Method:  r.Method,
		Path:    path,
		Query:   query,
		Headers: headers,
		Body:    string(body),
	}
	res, err := g.inv.Invoke(r.Context(), name, model.TriggerHTTP, ev)
	if err != nil {
		logger.Warn(r.Context(), "gateway invoke error", "fn", name, "err", err, "route_ms", routeMS)
		status := model.HTTPStatus(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"code":"` + model.CodeOf(err) + `","message":"` + escape(model.MessageOf(err)) + `"}`))
		return
	}
	if g.rec != nil {
		g.rec.Record(&model.Function{Name: name}, model.TriggerHTTP, res)
	}
	for k, v := range res.Headers {
		if strings.EqualFold(k, "content-length") {
			continue
		}
		w.Header().Set(k, v)
	}
	w.Header().Set("X-GSCF-Cold-Start", boolStr(res.ColdStart))
	w.Header().Set("X-GSCF-Wakeup-Ms", itoa(timeutil.DurationMS(res.Wakeup)))
	w.Header().Set("X-GSCF-Exec-Ms", itoa(timeutil.DurationMS(res.Exec)))
	w.Header().Set("X-GSCF-E2E-Ms", itoa(timeutil.DurationMS(res.E2E)))
	w.Header().Set("X-GSCF-Route-Ms", itoa(routeMS))
	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(res.Body)
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := v
	if n < 0 {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if v < 0 {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
