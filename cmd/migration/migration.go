package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Определяем флаги командной строки
	prefix := flag.String("prefix", "", "Префикс имени файла (короткий вариант: -p)")
	suffix := flag.String("suffix", "", "Суффикс имени файла (короткий вариант: -s)")
	timeFormat := flag.String("timeformat", "20060102150405", "Формат времени для имени файла")
	dir := flag.String("dir", "./migrations", "\tДиректория для сохранения файла миграции")
	s := flag.String("s", "", "") // Короткий вариант для suffix
	p := flag.String("p", "", "") // Короткий вариант для prefix

	// Настраиваем вывод справки по использованию
	flag.Usage = func() {
		fmt.Println("Использование: go run cmd/migration/migration.go [опции]")
		fmt.Println("Опции:")
		flag.VisitAll(func(f *flag.Flag) {
			if f.Usage != "" {
				fmt.Printf("  --%s\t%s (по умолчанию: \"%s\")\n", f.Name, f.Usage, f.DefValue)
			}
		})
	}
	flag.Parse()

	// Обрабатываем префикс
	finalPrefix := *prefix
	if finalPrefix == "" {
		finalPrefix = *p
	}
	if finalPrefix != "" {
		finalPrefix = finalPrefix + "_"
	}

	// Обрабатываем суффикс
	finalSuffix := *suffix
	if finalSuffix == "" {
		finalSuffix = *s
	}
	if finalSuffix != "" {
		finalSuffix = "_" + finalSuffix
	}

	// Формируем имя файла с временной меткой
	timestamp := time.Now().Format(*timeFormat)
	filename := fmt.Sprintf("%s%s%s_migration.sql", finalPrefix, timestamp, finalSuffix)
	filepath := filepath.Join(*dir, filename)

	// Проверяем, не существует ли уже файл с таким именем
	if _, err := os.Stat(filepath); err == nil {
		fmt.Printf("Файл с именем %s уже существует в директории %s. Выберите другой префикс, суффикс или директорию.\n", filename, *dir)
		return
	}

	// Создаем новый файл
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Printf("Ошибка при создании файла: %v. Убедитесь, что у вас есть права на запись в директорию %s и достаточно места на диске.\n", err, *dir)
		return
	}
	defer file.Close()

	fmt.Printf("Файл миграции успешно создан: %s\n", filepath)
}
