package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/google/uuid"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/zulfikorramatov/arche/generated/api"
)

type userAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (uuid.UUID, error)
}

func RequestValidator(swagger *openapi3.T, auth userAuthenticator) api.MiddlewareFunc {
	return nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, &nethttpmiddleware.Options{
		SilenceServersWarning: true,
		Options: openapi3filter.Options{
			AuthenticationFunc: func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
				username, password, ok := input.RequestValidationInput.Request.BasicAuth()
				if !ok {
					return errors.New("missing credentials")
				}
				if _, err := auth.Authenticate(ctx, username, password); err != nil {
					return errors.New("unauthorized")
				}
				return nil
			},
		},
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		},
	})
}
