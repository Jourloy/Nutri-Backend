package consent

import "time"

type ConsentRecord struct {
	Id           string    `db:"id" json:"id"`
	UserId       *string   `db:"user_id" json:"user_id,omitempty"`
	IpAddress    string    `db:"ip_address" json:"ip_address"`
	UserAgent    string    `db:"user_agent" json:"user_agent"`
	ConsentGiven bool      `db:"consent_given" json:"consent_given"`
	ConsentType  string    `db:"consent_type" json:"consent_type"`
	ConsentDate  time.Time `db:"consent_date" json:"consent_date"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type RecordConsentRequest struct {
	ConsentGiven bool   `json:"consent_given"`
	ConsentType  string `json:"consent_type,omitempty"`
}

type ConsentResponse struct {
	Success bool           `json:"success"`
	Record  *ConsentRecord `json:"record,omitempty"`
}
