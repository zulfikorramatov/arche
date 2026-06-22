package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.uber.org/zap"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/http/handler"
	appmiddleware "github.com/zulfikorramatov/arche/internal/http/middleware"
)

func NewRouter(log *zap.Logger, server *handler.Server) (http.Handler, error) {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.Logger(log))
	r.Use(appmiddleware.ErrorHandler())

	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("pong")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	// strictHandler adapts the typed StrictServerInterface to the low-level
	// chi handlers. The error funcs cover the cases the typed handlers can't
	// express: a body that fails to decode (400) and an error returned from a
	// handler (500).
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Error("handler error", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
		},
	})

	swagger, err := api.GetSpec()
	if err != nil {
		return nil, err
	}

	// The validator rejects requests that don't match the embedded spec
	// (missing params, malformed body, wrong content-type) before they reach a
	// handler. Servers stays set so the /api/v1 base path matches; the URL is
	// relative, so there is no real host check to silence beyond the warning.
	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, &nethttpmiddleware.Options{
		SilenceServersWarning: true,
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			writeJSONError(w, statusCode, message)
		},
	})

	return apmhttp.Wrap(api.HandlerWithOptions(strictHandler, api.ChiServerOptions{
		BaseURL:     "/api/v1",
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{validator},
	})), nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
