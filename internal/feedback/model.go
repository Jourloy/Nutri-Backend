package feedback

import "time"

// Feedback represents a single feedback submission or dismissal event.
type Feedback struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"userId" db:"user_id"`
	Status    string    `json:"status" db:"status"`
	Message   *string   `json:"message,omitempty" db:"message"`
	Viewed    bool      `json:"viewed" db:"viewed"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// StatusResponse describes whether feedback modal should be shown to the user.
type StatusResponse struct {
	ShouldShow    bool       `json:"shouldShow"`
	CooldownHours int        `json:"cooldownHours"`
	NextAllowedAt *time.Time `json:"nextAllowedAt,omitempty"`
	LastFeedback  *Feedback  `json:"lastFeedback,omitempty"`
}
