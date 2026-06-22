package handler

import (
	"context"
	"errors"

	"github.com/zulfikorramatov/arche/generated/api"
	"github.com/zulfikorramatov/arche/internal/domain"
)

func (s *Server) CreateUser(ctx context.Context, req api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	u, err := s.users.Create(ctx, string(req.Body.Email), req.Body.Name)
	if err != nil {
		if errors.Is(err, domain.ErrUserExists) {
			return api.CreateUser409JSONResponse{Error: "user already exists"}, nil
		}
		s.log.Error("create user", "error", err)
		return nil, err
	}
	return api.CreateUser201JSONResponse(toAPIUser(u)), nil
}

func (s *Server) GetUserByID(ctx context.Context, req api.GetUserByIDRequestObject) (api.GetUserByIDResponseObject, error) {
	u, err := s.users.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return api.GetUserByID404JSONResponse{Error: "user not found"}, nil
		}
		s.log.Error("get user", "error", err)
		return nil, err
	}
	return api.GetUserByID200JSONResponse(toAPIUser(u)), nil
}
