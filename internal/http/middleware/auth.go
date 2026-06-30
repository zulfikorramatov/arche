package middleware

import (
	"context"

	"github.com/google/uuid"
)

type AuthUser struct {
	ID       uuid.UUID
	Username string
}

type authCtxKey struct{}

func withAuthUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authCtxKey{}, user)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(authCtxKey{}).(AuthUser)
	return u, ok
}
