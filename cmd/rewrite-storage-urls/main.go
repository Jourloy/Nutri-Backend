package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/jourloy/somivyn/internal/storage"
)

type queryExecutor interface {
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type stringRow struct {
	ID    int64          `db:"id"`
	Value sql.NullString `db:"value"`
}

type blogContentRow struct {
	ID        int64  `db:"id"`
	ContentRu string `db:"content_ru"`
	ContentEn string `db:"content_en"`
}

type summaryRow struct {
	label   string
	updated int
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found")
	}

	databaseDSN := flag.String("database-dsn", os.Getenv("DATABASE_DSN"), "Postgres DSN")
	legacyEndpoint := flag.String("legacy-endpoint", os.Getenv("LEGACY_S3_ENDPOINT"), "Legacy S3 endpoint")
	legacyUseSSL := flag.Bool("legacy-use-ssl", boolEnv("LEGACY_S3_USE_SSL", false), "Use https for the legacy endpoint when scheme is omitted")
	targetEndpoint := flag.String("s3-endpoint", os.Getenv("S3_ENDPOINT"), "Target S3 endpoint")
	targetBucket := flag.String("s3-bucket-name", os.Getenv("S3_BUCKET_NAME"), "Target S3 bucket name")
	targetUseSSL := flag.Bool("s3-use-ssl", boolEnv("S3_USE_SSL", true), "Use https for the target S3 endpoint when scheme is omitted")
	dryRun := flag.Bool("dry-run", false, "Preview changes without updating the database")
	flag.Parse()

	if *databaseDSN == "" {
		exitf("DATABASE_DSN is required")
	}
	if *legacyEndpoint == "" {
		exitf("legacy endpoint is required")
	}
	if *targetEndpoint == "" {
		exitf("target S3 endpoint is required")
	}
	if *targetBucket == "" {
		exitf("target S3 bucket is required")
	}

	rewriter, err := storage.NewLegacyURLRewriter(*legacyEndpoint, *legacyUseSSL, storage.Config{
		Endpoint:   *targetEndpoint,
		BucketName: *targetBucket,
		UseSSL:     *targetUseSSL,
	})
	if err != nil {
		exitf("failed to build URL rewriter: %v", err)
	}

	db, err := sqlx.Connect("postgres", *databaseDSN)
	if err != nil {
		exitf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	var executor queryExecutor = db
	var tx *sqlx.Tx

	if !*dryRun {
		tx, err = db.BeginTxx(ctx, nil)
		if err != nil {
			exitf("failed to start transaction: %v", err)
		}
		defer tx.Rollback()
		executor = tx
	}

	summary, err := rewriteAll(ctx, executor, rewriter, *dryRun)
	if err != nil {
		exitf("rewrite failed: %v", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			exitf("failed to commit transaction: %v", err)
		}
	}

	mode := "apply"
	if *dryRun {
		mode = "dry-run"
	}
	fmt.Printf("Storage URL rewrite completed in %s mode\n", mode)

	total := 0
	for _, row := range summary {
		total += row.updated
		fmt.Printf("%s: %d\n", row.label, row.updated)
	}
	fmt.Printf("total: %d\n", total)
}

func rewriteAll(ctx context.Context, executor queryExecutor, rewriter *storage.LegacyURLRewriter, dryRun bool) ([]summaryRow, error) {
	summary := make([]summaryRow, 0, 9)

	urlColumns := []struct {
		label  string
		table  string
		column string
	}{
		{label: "ai_analysis_logs.image_url", table: "ai_analysis_logs", column: "image_url"},
		{label: "ai_violations.image_url", table: "ai_violations", column: "image_url"},
		{label: "blog_articles.preview_image_url", table: "blog_articles", column: "preview_image_url"},
		{label: "blog_articles.og_image_url", table: "blog_articles", column: "og_image_url"},
		{label: "recipe_books.og_image_url", table: "recipe_books", column: "og_image_url"},
		{label: "recipes.main_image_url", table: "recipes", column: "main_image_url"},
		{label: "recipes.og_image_url", table: "recipes", column: "og_image_url"},
		{label: "recipe_steps.image_url", table: "recipe_steps", column: "image_url"},
		{label: "recipe_images.image_url", table: "recipe_images", column: "image_url"},
	}

	for _, item := range urlColumns {
		updated, err := rewriteURLColumn(ctx, executor, rewriter, item.table, item.column, dryRun)
		if err != nil {
			return nil, err
		}
		summary = append(summary, summaryRow{label: item.label, updated: updated})
	}

	updatedMetadata, err := rewriteMetadataColumn(ctx, executor, rewriter, dryRun)
	if err != nil {
		return nil, err
	}
	summary = append(summary, summaryRow{label: "admin_notifications.metadata.imageUrl", updated: updatedMetadata})

	updatedContent, err := rewriteBlogContent(ctx, executor, rewriter, dryRun)
	if err != nil {
		return nil, err
	}
	summary = append(summary, summaryRow{label: "blog_articles.content_ru/content_en", updated: updatedContent})

	return summary, nil
}

func rewriteURLColumn(ctx context.Context, executor queryExecutor, rewriter *storage.LegacyURLRewriter, table, column string, dryRun bool) (int, error) {
	query := fmt.Sprintf("SELECT id, %s AS value FROM %s WHERE %s IS NOT NULL", column, table, column)

	var rows []stringRow
	if err := executor.SelectContext(ctx, &rows, query); err != nil {
		return 0, err
	}

	updated := 0
	updateQuery := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", table, column)
	for _, row := range rows {
		if !row.Value.Valid {
			continue
		}
		rewritten, changed := rewriter.RewriteURL(row.Value.String)
		if !changed {
			continue
		}
		updated++
		if dryRun {
			continue
		}
		if _, err := executor.ExecContext(ctx, updateQuery, rewritten, row.ID); err != nil {
			return 0, err
		}
	}

	return updated, nil
}

func rewriteMetadataColumn(ctx context.Context, executor queryExecutor, rewriter *storage.LegacyURLRewriter, dryRun bool) (int, error) {
	const query = `
		SELECT id, metadata::text AS value
		FROM admin_notifications
		WHERE notification_type = 'ai_violation' AND metadata IS NOT NULL`

	var rows []stringRow
	if err := executor.SelectContext(ctx, &rows, query); err != nil {
		return 0, err
	}

	updated := 0
	for _, row := range rows {
		if !row.Value.Valid {
			continue
		}
		rewritten, changed := rewriter.RewriteMetadata(row.Value.String)
		if !changed {
			continue
		}
		updated++
		if dryRun {
			continue
		}
		if _, err := executor.ExecContext(ctx, "UPDATE admin_notifications SET metadata = $1::jsonb WHERE id = $2", rewritten, row.ID); err != nil {
			return 0, err
		}
	}

	return updated, nil
}

func rewriteBlogContent(ctx context.Context, executor queryExecutor, rewriter *storage.LegacyURLRewriter, dryRun bool) (int, error) {
	const query = `
		SELECT id, content_ru, content_en
		FROM blog_articles
		WHERE deleted_at IS NULL`

	var rows []blogContentRow
	if err := executor.SelectContext(ctx, &rows, query); err != nil {
		return 0, err
	}

	updated := 0
	for _, row := range rows {
		nextRu, changedRu := rewriter.RewriteText(row.ContentRu)
		nextEn, changedEn := rewriter.RewriteText(row.ContentEn)
		if !changedRu && !changedEn {
			continue
		}
		updated++
		if dryRun {
			continue
		}
		if _, err := executor.ExecContext(
			ctx,
			"UPDATE blog_articles SET content_ru = $1, content_en = $2 WHERE id = $3",
			nextRu,
			nextEn,
			row.ID,
		); err != nil {
			return 0, err
		}
	}

	return updated, nil
}

func boolEnv(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func exitf(format string, args ...any) {
	fmt.Printf("ERROR: "+format+"\n", args...)
	os.Exit(1)
}
