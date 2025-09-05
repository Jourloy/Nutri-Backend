package achievement

import (
    "context"
    "encoding/json"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
    // Categories
    CreateCategory(ctx context.Context, c Category) (*Category, error)
    UpdateCategory(ctx context.Context, c Category) (*Category, error)
    DeleteCategory(ctx context.Context, id int64) error
    GetCategories(ctx context.Context) ([]Category, error)

    // Achievements
    CreateAchievement(ctx context.Context, a Achievement) (*Achievement, error)
    UpdateAchievement(ctx context.Context, a Achievement) (*Achievement, error)
    DeleteAchievement(ctx context.Context, id int64) error
    GetAchievements(ctx context.Context) ([]Achievement, error)

    // User achievements
    GetUserAchievementsMap(ctx context.Context, userId string) (map[int64]UserAchievement, error)
    InsertUserAchievement(ctx context.Context, userId string, achievementId int64) error
}

type repository struct { db *sqlx.DB }

func NewRepository() Repository { return &repository{db: database.Database} }

// ===== Categories =====
func (r *repository) CreateCategory(ctx context.Context, c Category) (*Category, error) {
    const q = `
        INSERT INTO achievement_categories (key, name, position)
        VALUES (:key, :name, :position)
        RETURNING id, key, name, position, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, c)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Category
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) UpdateCategory(ctx context.Context, c Category) (*Category, error) {
    const q = `
        UPDATE achievement_categories SET key=:key, name=:name, position=:position, updated_at=now()
        WHERE id=:id
        RETURNING id, key, name, position, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, c)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Category
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) DeleteCategory(ctx context.Context, id int64) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM achievement_categories WHERE id=$1`, id)
    return err
}

func (r *repository) GetCategories(ctx context.Context) ([]Category, error) {
    var res []Category
    if err := r.db.SelectContext(ctx, &res, `SELECT id, key, name, position, created_at, updated_at FROM achievement_categories ORDER BY position, id`); err != nil {
        return nil, err
    }
    return res, nil
}

// ===== Achievements =====
func (r *repository) CreateAchievement(ctx context.Context, a Achievement) (*Achievement, error) {
    // Marshal criteria
    b, _ := json.Marshal(a.Criteria)
    a.CriteriaRaw = b
    const q = `
      INSERT INTO achievements (key, category_id, name, description, icon, color, points, is_secret, enabled, prerequisite_id, criteria)
      VALUES (:key, :category_id, :name, :description, :icon, :color, :points, :is_secret, :enabled, :prerequisite_id, :criteria)
      RETURNING id, key, category_id, name, description, icon, color, points, is_secret, enabled, prerequisite_id, criteria, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, a)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Achievement
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    if len(out.CriteriaRaw) > 0 { _ = json.Unmarshal(out.CriteriaRaw, &out.Criteria) }
    return &out, nil
}

func (r *repository) UpdateAchievement(ctx context.Context, a Achievement) (*Achievement, error) {
    b, _ := json.Marshal(a.Criteria)
    a.CriteriaRaw = b
    const q = `
      UPDATE achievements
      SET key=:key, category_id=:category_id, name=:name, description=:description, icon=:icon, color=:color,
          points=:points, is_secret=:is_secret, enabled=:enabled, prerequisite_id=:prerequisite_id, criteria=:criteria,
          updated_at=now()
      WHERE id=:id
      RETURNING id, key, category_id, name, description, icon, color, points, is_secret, enabled, prerequisite_id, criteria, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, a)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Achievement
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    if len(out.CriteriaRaw) > 0 { _ = json.Unmarshal(out.CriteriaRaw, &out.Criteria) }
    return &out, nil
}

func (r *repository) DeleteAchievement(ctx context.Context, id int64) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM achievements WHERE id=$1`, id)
    return err
}

func (r *repository) GetAchievements(ctx context.Context) ([]Achievement, error) {
    const q = `
      SELECT id, key, category_id, name, description, icon, color, points, is_secret, enabled, prerequisite_id, criteria, created_at, updated_at
      FROM achievements
      WHERE deleted_at IS NULL
      ORDER BY id;`
    var res []Achievement
    if err := r.db.SelectContext(ctx, &res, q); err != nil { return nil, err }
    for i := range res {
        if len(res[i].CriteriaRaw) > 0 { _ = json.Unmarshal(res[i].CriteriaRaw, &res[i].Criteria) }
    }
    return res, nil
}

func (r *repository) GetUserAchievementsMap(ctx context.Context, userId string) (map[int64]UserAchievement, error) {
    const q = `SELECT id, user_id, achievement_id, achieved_at FROM user_achievements WHERE user_id=$1`
    var list []UserAchievement
    if err := r.db.SelectContext(ctx, &list, q, userId); err != nil { return nil, err }
    m := make(map[int64]UserAchievement, len(list))
    for _, ua := range list { m[ua.AchievementId] = ua }
    return m, nil
}

func (r *repository) InsertUserAchievement(ctx context.Context, userId string, achievementId int64) error {
    const q = `
      INSERT INTO user_achievements (user_id, achievement_id)
      VALUES ($1,$2)
      ON CONFLICT (user_id, achievement_id) DO NOTHING;`
    _, err := r.db.ExecContext(ctx, q, userId, achievementId)
    return err
}
