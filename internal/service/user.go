package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zulfikorramatov/arche/internal/domain"
)

type userRepository interface {
	List(ctx context.Context) ([]domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
}

type UserService struct {
	repo userRepository
}

func NewUserService(repo userRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *UserService) Authenticate(ctx context.Context, username, password string) (uuid.UUID, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return uuid.Nil, fmt.Errorf("invalid password: %w", err)
	}

	return user.ID, nil
}
