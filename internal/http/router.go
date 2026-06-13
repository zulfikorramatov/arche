package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/zulfikorramatov/arche/internal/http/handler"
	appmiddleware "github.com/zulfikorramatov/arche/internal/http/middleware"
)

func NewRouter(log *zap.Logger, userHandler *handler.UserHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.Logger(log))
	r.Use(appmiddleware.Recoverer(log))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Post("/", userHandler.Create)
			r.Get("/{id}", userHandler.GetByID)
		})
	})

	return r
}
