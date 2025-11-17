package user

import "context"

type ctxUserKeyType int

const ctxUserKey ctxUserKeyType = iota + 1

func ContextWithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxUserKey).(User)
	return u, ok
}
