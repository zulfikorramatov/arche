package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zulfikorramatov/arche/internal/domain"
	"github.com/zulfikorramatov/arche/internal/http/handler"
)

type fakeService struct {
	createErr error
	getErr    error
	user      *domain.User
}

func (f *fakeService) Create(_ context.Context, email, name string) (*domain.User, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &domain.User{ID: uuid.New(), Email: email, Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (f *fakeService) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &domain.User{ID: id, Email: "a@b.c", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func newTestRouter(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	r, err := NewRouter(zap.NewNop(), handler.NewServer(svc, zap.NewNop()))
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return r
}

func TestRouter_routesAndValidation(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		svc      *fakeService
		wantCode int
	}{
		{name: "healthz", method: http.MethodGet, path: "/healthz", svc: &fakeService{}, wantCode: http.StatusOK},
		{name: "create ok", method: http.MethodPost, path: "/api/v1/users", body: `{"email":"a@b.c","name":"bob"}`, svc: &fakeService{}, wantCode: http.StatusCreated},
		{name: "create invalid body rejected by validator", method: http.MethodPost, path: "/api/v1/users", body: `{"email":"a@b.c"}`, svc: &fakeService{}, wantCode: http.StatusBadRequest},
		{name: "create duplicate", method: http.MethodPost, path: "/api/v1/users", body: `{"email":"a@b.c","name":"bob"}`, svc: &fakeService{createErr: domain.ErrUserExists}, wantCode: http.StatusConflict},
		{name: "get invalid uuid rejected by validator", method: http.MethodGet, path: "/api/v1/users/not-a-uuid", svc: &fakeService{}, wantCode: http.StatusBadRequest},
		{name: "get ok", method: http.MethodGet, path: "/api/v1/users/" + uuid.New().String(), svc: &fakeService{}, wantCode: http.StatusOK},
		{name: "get not found", method: http.MethodGet, path: "/api/v1/users/" + uuid.New().String(), svc: &fakeService{getErr: domain.ErrUserNotFound}, wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, tt.svc)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("got status %d, want %d (body: %s)", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}
