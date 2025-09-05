package body

import (
    "context"
    "time"

    "github.com/charmbracelet/log"
    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

var (
    bgLogger = log.WithPrefix("[bodyw]")
)

func StartWorker() {
    go func() {
        // initial run after startup delay
        time.Sleep(5 * time.Second)
        runOnce()
        ticker := time.NewTicker(24 * time.Hour)
        defer ticker.Stop()
        for range ticker.C { runOnce() }
    }()
}

func runOnce() {
    svc := NewService()
    db := database.Database
    ids, err := getAllUserIds(db)
    if err != nil { bgLogger.Error("load users", "err", err); return }
    for _, uid := range ids {
        if _, err := svc.EvaluatePlateau(context.Background(), uid); err != nil {
            bgLogger.Warn("eval plateau", "user", uid, "err", err)
        }
    }
}

func getAllUserIds(db *sqlx.DB) ([]string, error) {
    rows, err := db.Queryx(`SELECT id FROM users WHERE deleted_at IS NULL`)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []string
    for rows.Next() { var id string; if err := rows.Scan(&id); err != nil { return nil, err }; res = append(res, id) }
    return res, rows.Err()
}

