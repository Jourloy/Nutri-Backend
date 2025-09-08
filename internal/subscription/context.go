package subscription

import "context"

type SubInfo struct {
    PlanType   string
    PlanCode   string
    Status     string
}

type ctxSubKeyType int

const ctxSubKey ctxSubKeyType = iota + 1

func ContextWithSubscription(ctx context.Context, si SubInfo) context.Context {
    return context.WithValue(ctx, ctxSubKey, si)
}

func SubscriptionFromContext(ctx context.Context) (SubInfo, bool) {
    v, ok := ctx.Value(ctxSubKey).(SubInfo)
    return v, ok
}

