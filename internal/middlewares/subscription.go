package middlewares

import (
    "database/sql"
    "net/http"
    "time"

    "github.com/jourloy/nutri-backend/internal/auth"
    "github.com/jourloy/nutri-backend/internal/database"
    "github.com/jourloy/nutri-backend/internal/subscription"
)

// Subscription middleware loads current user's subscription/plan info
// and injects it into request context.
func Subscription(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Must have user
        u, ok := auth.UserFromContext(r.Context())
        if !ok || u.Id == "" {
            next.ServeHTTP(w, r)
            return
        }

        // Load latest subscription + plan_type
        type row struct{
            PlanType string `db:"plan_type"`
            PlanCode string `db:"code"`
            Status   string `db:"status"`
            PeriodEnd *time.Time `db:"period_end"`
            TrialEnd  *time.Time `db:"trial_end"`
        }

        var out row
        err := database.Database.GetContext(r.Context(), &out, `
            SELECT p.plan_type, p.code, s.status, s.period_end, s.trial_end
            FROM subscriptions s
            JOIN plans p ON p.id = s.plan_id
            WHERE s.user_id = $1
            ORDER BY s.created_at DESC
            LIMIT 1`, u.Id)

        si := subscription.SubInfo{ PlanType: "START", PlanCode: "START", Status: "none" }
        if err == nil {
            // Consider expired subscriptions as START
            active := true
            now := time.Now()
            if out.PeriodEnd != nil && out.PeriodEnd.Before(now) { active = false }
            if out.TrialEnd != nil && out.TrialEnd.Before(now) && out.Status == "trialing" { active = false }
            if active {
                si.PlanType = out.PlanType
                si.PlanCode = out.PlanCode
                si.Status = out.Status
            }
        } else if err != sql.ErrNoRows {
            // On DB error: fall back silently to START
        }

        ctx := subscription.ContextWithSubscription(r.Context(), si)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

