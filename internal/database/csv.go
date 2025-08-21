package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type ImportStats struct {
	Total    int
	Inserted int
	Skipped  int
	Errors   int
}

type CSVRepository struct {
	DB *sqlx.DB
}

// ImportTemplatesFromCSV читает CSV со столбцами:
// "Продукт;Белки;Жиры;Углеводы;Ккал" (порядок как в примере).
// Если запись с таким name уже есть — пропускаем (ON CONFLICT DO NOTHING).
func (r *CSVRepository) ImportTemplatesFromCSV(ctx context.Context, csvReader io.Reader) (ImportStats, error) {
	var st ImportStats

	cr := csv.NewReader(csvReader)
	cr.Comma = ';'
	cr.FieldsPerRecord = -1 // допускаем лишние/неполные
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	// читаем заголовок
	header, err := cr.Read()
	if err != nil {
		return st, fmt.Errorf("read header: %w", err)
	}
	// нормализуем заголовки
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}

	nameIdx, protIdx, fatIdx, carbIdx, kcalIdx, err := detectColumns(header)
	if err != nil {
		return st, err
	}

	// транзакция + prepared stmt
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return st, err
	}
	defer func() {
		_ = tx.Rollback() // на случай раннего выхода; Commit ниже перетрёт
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO templates (name, calories, protein, fat, carbs, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5, $6, $6)
		ON CONFLICT (name) DO NOTHING
	`)
	if err != nil {
		return st, err
	}
	defer stmt.Close()

	now := time.Now()

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			st.Errors++
			continue
		}
		st.Total++

		// Груминг полей
		name := strings.TrimSpace(get(rec, nameIdx))
		if name == "" {
			st.Skipped++
			continue
		}

		// Парсим числа (заменим запятую на точку на всякий случай)
		parseFloat := func(s string) (float64, error) {
			s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
			if s == "" {
				return 0, nil
			}
			return strconv.ParseFloat(s, 64)
		}

		prot, err1 := parseFloat(get(rec, protIdx))
		fat, err2 := parseFloat(get(rec, fatIdx))
		carbs, err3 := parseFloat(get(rec, carbIdx))
		kcal, err4 := parseFloat(get(rec, kcalIdx))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			st.Errors++
			continue
		}

		// Вставка (дубликаты по name пропускаем)
		res, err := stmt.ExecContext(ctx, name, kcal, prot, fat, carbs, now)
		if err != nil {
			st.Errors++
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			st.Skipped++
		} else {
			st.Inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return st, err
	}
	return st, nil
}

// detectColumns — определяем индексы колонок по заголовку.
func detectColumns(header []string) (name, prot, fat, carb, kcal int, err error) {
	const notFound = -1
	name, prot, fat, carb, kcal = notFound, notFound, notFound, notFound, notFound

	for i, h := range header {
		key := strings.ToLower(strings.ReplaceAll(h, " ", ""))
		switch key {
		case "продукт", "название", "наименование":
			name = i
		case "белки", "бел,г", "бел":
			prot = i
		case "жиры", "жир,г", "жир":
			fat = i
		case "углеводы", "угл,г", "угл":
			carb = i
		case "ккал", "калории", "энергетическаяценность":
			kcal = i
		}
	}

	if name < 0 || prot < 0 || fat < 0 || carb < 0 || kcal < 0 {
		return -1, -1, -1, -1, -1, fmt.Errorf("не удалось распознать заголовки CSV: %v", header)
	}
	return
}

func get(rec []string, idx int) string {
	if idx >= 0 && idx < len(rec) {
		return rec[idx]
	}
	return ""
}
