package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zulfikorramatov/arche/internal/domain"
	"github.com/zulfikorramatov/arche/pkg/redis"
)

const userCacheTTL = 5 * time.Minute

type userRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type UserService struct {
	repo  userRepository
	cache *redis.Client
}

func NewUserService(repo userRepository, cache *redis.Client) *UserService {
	return &UserService{repo: repo, cache: cache}
}

func (s *UserService) Create(ctx context.Context, email, name string) (*domain.User, error) {
	now := time.Now().UTC()
	u := &domain.User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	key := userCacheKey(id)

	if data, err := s.cache.Get(ctx, key).Bytes(); err == nil {
		var cached domain.User
		if json.Unmarshal(data, &cached) == nil {
			return &cached, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		// cache miss is fine; other errors fall through to DB
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(u); err == nil {
		_ = s.cache.Set(ctx, key, data, userCacheTTL).Err()
	}
	return u, nil
}

func userCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("user:%s", id)
}
