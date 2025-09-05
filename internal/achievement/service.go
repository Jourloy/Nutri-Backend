package achievement

import (
    "context"
    "errors"
    "time"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

type Service interface {
    // Admin
    CreateCategory(ctx context.Context, c Category) (*Category, error)
    UpdateCategory(ctx context.Context, c Category) (*Category, error)
    DeleteCategory(ctx context.Context, id int64) error
    GetCategories(ctx context.Context) ([]Category, error)

    CreateAchievement(ctx context.Context, a Achievement) (*Achievement, error)
    UpdateAchievement(ctx context.Context, a Achievement) (*Achievement, error)
    DeleteAchievement(ctx context.Context, id int64) error

    // User
    ListForUser(ctx context.Context, userId string) ([]AchievementView, error)
    ListUserUnlocked(ctx context.Context, userId string) ([]AchievementView, error)
    EvaluateUser(ctx context.Context, userId string) ([]AchievementView, error)
}

type service struct {
    repo Repository
    db   *sqlx.DB
}

func NewService() Service { return &service{repo: NewRepository(), db: database.Database} }

// ===== Admin passthrough =====
func (s *service) CreateCategory(ctx context.Context, c Category) (*Category, error) { return s.repo.CreateCategory(ctx, c) }
func (s *service) UpdateCategory(ctx context.Context, c Category) (*Category, error) { return s.repo.UpdateCategory(ctx, c) }
func (s *service) DeleteCategory(ctx context.Context, id int64) error { return s.repo.DeleteCategory(ctx, id) }
func (s *service) GetCategories(ctx context.Context) ([]Category, error) { return s.repo.GetCategories(ctx) }

func (s *service) CreateAchievement(ctx context.Context, a Achievement) (*Achievement, error) { return s.repo.CreateAchievement(ctx, a) }
func (s *service) UpdateAchievement(ctx context.Context, a Achievement) (*Achievement, error) { return s.repo.UpdateAchievement(ctx, a) }
func (s *service) DeleteAchievement(ctx context.Context, id int64) error { return s.repo.DeleteAchievement(ctx, id) }

// ===== User facing =====
func (s *service) ListForUser(ctx context.Context, userId string) ([]AchievementView, error) {
    achs, err := s.repo.GetAchievements(ctx)
    if err != nil { return nil, err }
    cats, _ := s.repo.GetCategories(ctx)
    unlocked, err := s.repo.GetUserAchievementsMap(ctx, userId)
    if err != nil { return nil, err }

    // Build category lookup
    catById := map[int64]Category{}
    for _, c := range cats { catById[c.Id] = c }

    res := make([]AchievementView, 0, len(achs))
    for _, a := range achs {
        if !a.Enabled { continue }
        // prerequisites
        if a.PrerequisiteId != nil {
            if _, ok := unlocked[*a.PrerequisiteId]; !ok {
                // still locked by prerequisite: mark progress but keep locked
            }
        }
        var current float64
        cur, err := s.computeMetric(ctx, userId, a.Criteria)
        if err == nil { current = cur }
        uv := AchievementView{
            Id: a.Id,
            Key: a.Key,
            Name: a.Name,
            Description: a.Description,
            Icon: a.Icon,
            Color: a.Color,
            Points: a.Points,
            IsSecret: a.IsSecret,
            Enabled: a.Enabled,
            Requirement: a.Criteria.Threshold,
            Current: current,
            Unlocked: false,
        }
        if a.CategoryId != nil {
            if c, ok := catById[*a.CategoryId]; ok {
                uv.CategoryId = &c.Id
                uv.Category = &c.Name
                uv.CategoryKey = &c.Key
            }
        }
        if _, ok := unlocked[a.Id]; ok {
            uv.Unlocked = true
        } else {
            // hide secret details
            if a.IsSecret {
                uv.Description = "Секретная награда"
                uv.Requirement = 1
                uv.Current = 0
            }
        }
        res = append(res, uv)
    }
    return res, nil
}

func (s *service) ListUserUnlocked(ctx context.Context, userId string) ([]AchievementView, error) {
    achs, err := s.repo.GetAchievements(ctx)
    if err != nil { return nil, err }
    unlocked, err := s.repo.GetUserAchievementsMap(ctx, userId)
    if err != nil { return nil, err }
    cats, _ := s.repo.GetCategories(ctx)
    catById := map[int64]Category{}
    for _, c := range cats { catById[c.Id] = c }
    res := []AchievementView{}
    for _, a := range achs {
        if _, ok := unlocked[a.Id]; !ok { continue }
        cur, _ := s.computeMetric(ctx, userId, a.Criteria)
        uv := AchievementView{
            Id: a.Id,
            Key: a.Key,
            Name: a.Name,
            Description: a.Description,
            Icon: a.Icon,
            Color: a.Color,
            Points: a.Points,
            IsSecret: a.IsSecret,
            Enabled: a.Enabled,
            Requirement: a.Criteria.Threshold,
            Current: cur,
            Unlocked: true,
        }
        if a.CategoryId != nil {
            if c, ok := catById[*a.CategoryId]; ok {
                uv.CategoryId = &c.Id
                uv.Category = &c.Name
                uv.CategoryKey = &c.Key
            }
        }
        res = append(res, uv)
    }
    return res, nil
}

func (s *service) EvaluateUser(ctx context.Context, userId string) ([]AchievementView, error) {
    achs, err := s.repo.GetAchievements(ctx)
    if err != nil { return nil, err }
    unlocked, err := s.repo.GetUserAchievementsMap(ctx, userId)
    if err != nil { return nil, err }
    cats, _ := s.repo.GetCategories(ctx)
    catById := map[int64]Category{}
    for _, c := range cats { catById[c.Id] = c }

    var newly []AchievementView
    for _, a := range achs {
        if !a.Enabled { continue }
        // Already unlocked?
        if _, ok := unlocked[a.Id]; ok { continue }
        // Prerequisite check
        if a.PrerequisiteId != nil {
            if _, ok := unlocked[*a.PrerequisiteId]; !ok { continue }
        }
        cur, err := s.computeMetric(ctx, userId, a.Criteria)
        if err != nil { continue }
        if cur >= a.Criteria.Threshold {
            // award
            if err := s.repo.InsertUserAchievement(ctx, userId, a.Id); err == nil {
                uv := AchievementView{
                    Id: a.Id,
                    Key: a.Key,
                    Name: a.Name,
                    Description: a.Description,
                    Icon: a.Icon,
                    Color: a.Color,
                    Points: a.Points,
                    IsSecret: a.IsSecret,
                    Enabled: a.Enabled,
                    Requirement: a.Criteria.Threshold,
                    Current: cur,
                    Unlocked: true,
                }
                if a.CategoryId != nil { if c, ok := catById[*a.CategoryId]; ok { uv.CategoryId = &c.Id; uv.Category = &c.Name; uv.CategoryKey = &c.Key } }
                newly = append(newly, uv)
            }
        }
    }
    return newly, nil
}

// ===== metrics =====
func (s *service) computeMetric(ctx context.Context, userId string, c Criteria) (float64, error) {
    switch c.Metric {
    case "total_products_count":
        var n int64
        err := s.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM products WHERE user_id=$1`, userId)
        return float64(n), err
    case "today_products_count":
        var n int64
        err := s.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM products WHERE user_id=$1 AND created_at >= CURRENT_DATE AND created_at < CURRENT_DATE + INTERVAL '1 day'`, userId)
        return float64(n), err
    case "total_calories_sum":
        var v *float64
        err := s.db.GetContext(ctx, &v, `SELECT COALESCE(SUM(calories),0)::float FROM products WHERE user_id=$1`, userId)
        if v == nil { zero := 0.0; v = &zero }
        return *v, err
    case "total_protein_sum":
        var v *float64
        err := s.db.GetContext(ctx, &v, `SELECT COALESCE(SUM(protein),0)::float FROM products WHERE user_id=$1`, userId)
        if v == nil { zero := 0.0; v = &zero }
        return *v, err
    case "daily_streak_products":
        // current consecutive days with at least 1 product, including today if applicable
        return float64(s.currentStreakDays(ctx, userId)), nil
    default:
        return 0, errors.New("unknown metric: " + c.Metric)
    }
}

func (s *service) currentStreakDays(ctx context.Context, userId string) int {
    // get all days where there are events (products)
    // limit last 120 days for performance
    type row struct { D time.Time `db:"d"` }
    var days []row
    _ = s.db.SelectContext(ctx, &days, `
        SELECT DISTINCT date_trunc('day', created_at)::date AS d
        FROM products
        WHERE user_id=$1
        AND created_at >= CURRENT_DATE - INTERVAL '120 day'
        ORDER BY d DESC`, userId)
    if len(days) == 0 { return 0 }
    // make a set of dates
    m := map[string]struct{}{}
    for _, r := range days { m[r.D.Format("2006-01-02")] = struct{}{} }
    // iterate backward from today
    streak := 0
    for i := 0; i < 365; i++ {
        d := time.Now().AddDate(0,0,-i).Format("2006-01-02")
        if _, ok := m[d]; ok { streak++ } else { break }
    }
    return streak
}
