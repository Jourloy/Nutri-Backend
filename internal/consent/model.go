package consent

import "time"

const (
	TypePersonalDataProcessing = "personal_data_processing"
	TypeAnalyticsCookies       = "analytics_cookies"
	TypeMarketingAds           = "marketing_ads"
	TypeSpecialCategoryHealth  = "special_category_health"
	TypePublicDistribution     = "public_distribution"

	DefaultDocumentVersion = "2026-05-01"
)

type ConsentRecord struct {
	Id              string    `db:"id" json:"id"`
	UserId          *string   `db:"user_id" json:"user_id,omitempty"`
	IpAddress       string    `db:"ip_address" json:"ip_address"`
	UserAgent       string    `db:"user_agent" json:"user_agent"`
	ConsentGiven    bool      `db:"consent_given" json:"consent_given"`
	ConsentType     string    `db:"consent_type" json:"consent_type"`
	DocumentVersion string    `db:"document_version" json:"document_version"`
	Locale          string    `db:"locale" json:"locale"`
	Source          string    `db:"source" json:"source"`
	ConsentDate     time.Time `db:"consent_date" json:"consent_date"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type RecordConsentRequest struct {
	ConsentGiven    bool   `json:"consent_given"`
	ConsentType     string `json:"consent_type,omitempty"`
	DocumentVersion string `json:"document_version,omitempty"`
	Locale          string `json:"locale,omitempty"`
	Source          string `json:"source,omitempty"`
}

type ConsentResponse struct {
	Success bool           `json:"success"`
	Record  *ConsentRecord `json:"record,omitempty"`
}

type ConsentStatusResponse struct {
	Success bool                     `json:"success"`
	Records map[string]ConsentRecord `json:"records"`
}
