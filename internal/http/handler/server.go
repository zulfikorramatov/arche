package handler

import (
	"context"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/entity"
	"github.com/zulfikorramatov/arche/pkg/logger"
)

var _ api.StrictServerInterface = (*Server)(nil)

type userService interface {
	List(ctx context.Context) ([]entity.User, error)
}

type Server struct {
	log     *logger.Logger
	userSvc userService
}

func NewServer(log *logger.Logger, userSvc userService) *Server {
	return &Server{log: log, userSvc: userSvc}
}

func (s *Server) ListUsers(ctx context.Context, _ api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	users, err := s.userSvc.List(ctx)
	if err != nil {
		s.log.Error("list users", "error", err)
		return api.ListUsers500JSONResponse{Error: "internal error"}, nil
	}

	result := make(api.UserList, 0, len(users))
	for _, u := range users {
		result = append(result, api.User{
			Id:        u.ID,
			Username:  u.Username,
			CreatedAt: u.CreatedAt,
		})
	}

	return api.ListUsers200JSONResponse(result), nil
}
