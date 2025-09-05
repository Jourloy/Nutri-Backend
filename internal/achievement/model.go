package achievement

import "time"

type Category struct {
    Id        int64     `json:"id" db:"id"`
    Key       string    `json:"key" db:"key"`
    Name      string    `json:"name" db:"name"`
    Position  int64     `json:"position" db:"position"`
    CreatedAt time.Time `json:"-" db:"created_at"`
    UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type Criteria struct {
    // Metric defines what to measure. Supported: total_products_count, today_products_count,
    // total_calories_sum, total_protein_sum, daily_streak_products
    Metric      string   `json:"metric"`
    Threshold   float64  `json:"threshold"`
    WindowDays  *int     `json:"windowDays,omitempty"`
    Consecutive bool     `json:"consecutive,omitempty"`
}

type Achievement struct {
    Id             int64      `json:"id" db:"id"`
    Key            string     `json:"key" db:"key"`
    CategoryId     *int64     `json:"categoryId,omitempty" db:"category_id"`
    Name           string     `json:"name" db:"name"`
    Description    string     `json:"description" db:"description"`
    Icon           *string    `json:"icon,omitempty" db:"icon"`
    Color          *string    `json:"color,omitempty" db:"color"`
    Points         int        `json:"points" db:"points"`
    IsSecret       bool       `json:"isSecret" db:"is_secret"`
    Enabled        bool       `json:"enabled" db:"enabled"`
    PrerequisiteId *int64     `json:"prerequisiteId,omitempty" db:"prerequisite_id"`
    Criteria       Criteria   `json:"criteria" db:"-"`
    CriteriaRaw    []byte     `json:"-" db:"criteria"`
    CreatedAt      time.Time  `json:"-" db:"created_at"`
    UpdatedAt      time.Time  `json:"-" db:"updated_at"`
}

type UserAchievement struct {
    Id            int64     `json:"id" db:"id"`
    UserId        string    `json:"userId" db:"user_id"`
    AchievementId int64     `json:"achievementId" db:"achievement_id"`
    AchievedAt    time.Time `json:"achievedAt" db:"achieved_at"`
}

// Response view types
type AchievementView struct {
    Id          int64    `json:"id"`
    Key         string   `json:"key"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Icon        *string  `json:"icon,omitempty"`
    Color       *string  `json:"color,omitempty"`
    Points      int      `json:"points"`
    IsSecret    bool     `json:"isSecret"`
    Enabled     bool     `json:"enabled"`
    CategoryId  *int64   `json:"categoryId,omitempty"`
    CategoryKey *string  `json:"categoryKey,omitempty"`
    Category    *string  `json:"categoryName,omitempty"`
    Requirement float64  `json:"requirement"`
    Current     float64  `json:"current"`
    Unlocked    bool     `json:"unlocked"`
}

