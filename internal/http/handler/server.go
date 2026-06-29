package handler

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/domain"
	"github.com/zulfikorramatov/arche/pkg/logger"
)

var _ api.StrictServerInterface = (*Server)(nil)

type userService interface {
	Create(ctx context.Context, email, name string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Server struct {
	users userService
	log   *logger.Logger
}

func NewServer(users userService, log *logger.Logger) *Server {
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
