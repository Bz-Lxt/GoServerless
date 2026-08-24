package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/store"
	"github.com/gogo/goserverless/internal/timeutil"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = idgen.RequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := logger.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		logger.Info(r.Context(), "http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"ms", timeutil.DurationMS(time.Since(start)),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type Auth struct {
	st       *store.Store
	user     string
	pass     string
	skipPref []string
}

func NewAuth(st *store.Store, user, pass string) *Auth {
	return &Auth{st: st, user: user, pass: pass, skipPref: []string{"/api/v1/health", "/api/v1/auth/login", "/api/v1/run/"}}
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for _, p := range a.skipPref {
			if strings.HasPrefix(path, p) {
				next.ServeHTTP(w, r)
				return
			}
		}
		token := bearer(r.Header.Get("Authorization"))
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeErr(w, errUnauthorized())
			return
		}
		user, exp, err := a.st.GetSession(r.Context(), token)
		if err != nil || user == "" || time.Now().After(exp) {
			writeErr(w, errUnauthorized())
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearer(h string) string {
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func errUnauthorized() error {
	return newUnauthorized()
}
