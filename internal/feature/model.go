package feature

// Feature represents a feature record.
type Feature struct {
	Key         string `json:"key" db:"key"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Unit        string `json:"unit" db:"unit"`
}
