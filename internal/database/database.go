package database

import (
	"database/sql"
	"os"

	"github.com/charmbracelet/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/jourloy/nutri-backend/internal/lib"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: `[rpst]`,
		Level:  log.DebugLevel,
	})

	Database *sqlx.DB
)

// Connect to DB and store it in public var
func Connect() {
	db, err := sqlx.Connect("postgres", lib.Config.DatabaseDSN)
	if err != nil {
		logger.Fatal("Cannot connect to database", "error", err)
	}

	Database = db

	Migrate(db)
}

func Migrate(db *sqlx.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := db.Exec(query)
	if err != nil {
		logger.Fatal("Cannot create migrations table", "error", err)
	}

	files, err := os.ReadDir("migrations")
	if err != nil {
		logger.Fatal("Cannot read migrations directory", "error", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			content, err := os.ReadFile("migrations/" + file.Name())
			if err != nil {
				logger.Fatal("Cannot read migration file", "file", file.Name(), "error", err)
			}

			query := "SELECT id FROM migrations WHERE name = $1"
			var id int
			err = db.Get(&id, query, file.Name())
			if err != nil && err != sql.ErrNoRows {
				logger.Fatal("Cannot get migration id", "file", file.Name(), "error", err)
			}

			if id != 0 {
				continue
			}

			_, err = db.Exec(string(content))
			if err != nil {
				logger.Fatal("Cannot execute migration", "file", file.Name(), "error", err)
			}

			logger.Info("Migration executed successfully", "file", file.Name())
			query = "INSERT INTO migrations (name) VALUES ($1)"
			_, err = db.Exec(query, file.Name())
			if err != nil {
				logger.Fatal("Cannot insert migration", "file", file.Name(), "error", err)
			}
		}
	}

	logger.Info("Migrations executed successfully")
}
