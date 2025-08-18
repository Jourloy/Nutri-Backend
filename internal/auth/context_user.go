package auth

import (
	"context"

	"github.com/jourloy/nutri-backend/internal/user"
)

type ctxUserKeyType int

const ctxUserKey ctxUserKeyType = iota + 1

func ContextWithUser(ctx context.Context, u user.User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (user.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(user.User)
	return u, ok
}
