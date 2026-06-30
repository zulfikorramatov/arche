package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type userAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (uuid.UUID, error)
}

func BasicAuth(auth userAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)
				return
			}

			id, err := auth.Authenticate(r.Context(), username, password)
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := withAuthUser(r.Context(), AuthUser{ID: id, Username: username})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
