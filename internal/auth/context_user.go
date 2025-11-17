package auth

import (
	"context"

	"github.com/jourloy/nutri-backend/internal/user"
)

// ContextWithUser is a wrapper around user.ContextWithUser for backward compatibility
func ContextWithUser(ctx context.Context, u user.User) context.Context {
	return user.ContextWithUser(ctx, u)
}

// UserFromContext is a wrapper around user.UserFromContext for backward compatibility
func UserFromContext(ctx context.Context) (user.User, bool) {
	return user.UserFromContext(ctx)
}
