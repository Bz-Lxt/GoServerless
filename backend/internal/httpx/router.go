package httpx

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/gogo/goserverless/internal/gateway"
	"github.com/gogo/goserverless/internal/handler"
)

func NewRouter(api *handler.API, authMW *handler.Auth, authAPI *handler.AuthAPI, gw *gateway.Gateway) http.Handler {
	r := chi.NewRouter()
	r.Use(handler.RequestID)
	r.Use(handler.AccessLog)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-GSCF-Cold-Start", "X-GSCF-Wakeup-Ms", "X-GSCF-Exec-Ms", "X-GSCF-E2E-Ms"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/v1/health", handler.Health)
	r.Post("/api/v1/auth/login", authAPI.Login)

	r.Group(func(r chi.Router) {
		r.Use(authMW.Middleware)
		r.Post("/api/v1/auth/logout", authAPI.Logout)
		r.Get("/api/v1/auth/me", authAPI.Me)
		r.Get("/api/v1/overview", api.Overview)
		r.Get("/api/v1/templates", api.Templates)
		r.Get("/api/v1/functions", api.List)
		r.Post("/api/v1/functions", api.Create)
		r.Get("/api/v1/functions/{name}", api.Get)
		r.Patch("/api/v1/functions/{name}", api.Update)
		r.Delete("/api/v1/functions/{name}", api.Delete)
		r.Post("/api/v1/functions/{name}/deploy", api.Deploy)
		r.Post("/api/v1/functions/{name}/rollback", api.Rollback)
		r.Get("/api/v1/functions/{name}/versions", api.Versions)
		r.Get("/api/v1/functions/{name}/triggers", api.Triggers)
		r.Put("/api/v1/functions/{name}/triggers", api.PutTriggers)
		r.Get("/api/v1/functions/{name}/metrics", api.Metrics)
		r.Get("/api/v1/functions/{name}/invocations", api.Invocations)
		r.Get("/api/v1/functions/{name}/logs/stream", api.Logs)
	})

	r.HandleFunc("/api/v1/run/{function_name}", gw.ServeHTTP)
	r.HandleFunc("/api/v1/run/{function_name}/*", gw.ServeHTTP)
	return r
}

func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 8 * time.Second,
		ReadTimeout:       35 * time.Second,
		WriteTimeout:      0, // SSE
		IdleTimeout:       60 * time.Second,
	}
}
