package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.elastic.co/apm/module/apmhttp/v2"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/http/handler"
	appmiddleware "github.com/zulfikorramatov/arche/internal/http/middleware"
	"github.com/zulfikorramatov/arche/pkg/logger"
)

type userAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (uuid.UUID, error)
}

func NewRouter(
	log *logger.Logger,
	server *handler.Server,
	auth userAuthenticator,
) (http.Handler, error) {
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Error("handler error", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
		},
	})

	swagger, err := api.GetSpec()
	if err != nil {
		return nil, err
	}

	return apmhttp.Wrap(api.HandlerWithOptions(strictHandler, api.ChiServerOptions{
		BaseRouter: chi.NewMux(),
		Middlewares: []api.MiddlewareFunc{
			appmiddleware.RequestValidator(swagger, auth),
			chimiddleware.RequestID,
			appmiddleware.Logger(log),
			appmiddleware.ErrorHandler(),
		},
	})), nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
