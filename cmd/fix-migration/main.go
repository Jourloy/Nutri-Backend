package main

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found")
	}

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		fmt.Println("ERROR: DATABASE_DSN not set")
		os.Exit(1)
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		fmt.Printf("ERROR: Cannot connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("✅ Connected to database")

	// Step 1: Remove failed migration record
	fmt.Println("🧹 Cleaning up failed migration record...")
	_, err = db.Exec("DELETE FROM migrations WHERE name = '20260128120003_add_supplement_callback_mappings.sql'")
	if err != nil {
		fmt.Printf("ERROR: Cannot delete migration record: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Migration record removed")

	// Step 2: Drop table if exists
	fmt.Println("🧹 Dropping table if exists...")
	_, err = db.Exec("DROP TABLE IF EXISTS supplement_callback_mappings")
	if err != nil {
		fmt.Printf("ERROR: Cannot drop table: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Table dropped")

	// Step 3: Apply migration
	fmt.Println("📝 Applying migration...")
	migrationSQL, err := os.ReadFile("migrations/20260128120003_add_supplement_callback_mappings.sql")
	if err != nil {
		fmt.Printf("ERROR: Cannot read migration file: %v\n", err)
		os.Exit(1)
	}

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		fmt.Printf("ERROR: Cannot execute migration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Migration executed successfully")

	// Step 4: Record migration
	fmt.Println("📝 Recording migration...")
	_, err = db.Exec("INSERT INTO migrations (name) VALUES ('20260128120003_add_supplement_callback_mappings.sql')")
	if err != nil {
		fmt.Printf("ERROR: Cannot record migration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Migration recorded")

	fmt.Println("\n🎉 All done! You can now restart the Telegram bot.")
}
