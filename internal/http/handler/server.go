package handler

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/domain"
)

// Server implements the generated StrictServerInterface. The strict adapter
// parses requests and serializes responses, so methods here only translate
// between the generated request/response types and the service layer.
var _ api.StrictServerInterface = (*Server)(nil)

type userService interface {
	Create(ctx context.Context, email, name string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Server struct {
	users userService
	log   *zap.Logger
}

func NewServer(users userService, log *zap.Logger) *Server {
	return &Server{users: users, log: log}
}

func toAPIUser(u *domain.User) api.User {
	return api.User{
		Id:        u.ID,
		Email:     openapi_types.Email(u.Email),
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
