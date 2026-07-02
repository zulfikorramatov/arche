package repository

import (
	"context"
	"fmt"

	"github.com/zulfikorramatov/arche/internal/entity"
	"github.com/zulfikorramatov/arche/pkg/postgres"
)

type UserRepository struct {
	pool *postgres.Pool
}

func NewUserRepository(pool *postgres.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) List(ctx context.Context) ([]entity.User, error) {
	const q = `SELECT id, username, password, created_at FROM users`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return users, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	const q = `SELECT id, username, password, created_at FROM users WHERE username = $1`

	var u entity.User
	err := r.pool.QueryRow(ctx, q, username).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}

	return &u, nil
}
