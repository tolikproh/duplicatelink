package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tolikproh/duplicatelink/internal/cleaner"
	"github.com/tolikproh/duplicatelink/internal/restore"
	"github.com/tolikproh/duplicatelink/internal/scanner"
)

func main() {
	// Флаги
	hashAlg := flag.String("hash", "sha1", "Алгоритм хеширования: sha1 или md5")
	mode := flag.String("mode", "scan", "Режим работы: scan (сканирование), clean (очистка) или restore (восстановление)")
	jsonFile := flag.String("json", "", "Путь к JSON файлу с результатами сканирования (для режима clean)")
	outputFile := flag.String("output", "duplicates.json", "Путь для сохранения результатов сканирования в JSON")
	includeHidden := flag.Bool("include-hidden", false, "Включить сканирование скрытых папок")
	scanReport := flag.String("scan-report", "scan_report.md", "Путь для сохранения Markdown отчета сканирования")
	targetDir := flag.String("target", "", "Папка назначения для перемещения файлов (для режима clean)")
	restoreFile := flag.String("restore", "", "Путь к JSON файлу восстановления (для режима restore)")
	workers := flag.Int("workers", 3, "Количество воркеров для параллельного хеширования (3-10)")
	flag.Parse()

	// Если нет аргументов - выводим справку
	if len(os.Args) == 1 {
		fmt.Println("DuplicateLink - утилита для поиска и удаления дубликатов файлов")
		fmt.Println()
		fmt.Println("Использование:")
		fmt.Println("  duplicatelink [флаги] [путь]")
		fmt.Println()
		fmt.Println("Режимы работы:")
		fmt.Println("  scan    - Поиск дубликатов (по умолчанию)")
		fmt.Println("  clean   - Очистка дубликатов")
		fmt.Println("  restore - Восстановление из бэкапа")
		fmt.Println()
		fmt.Println("Флаги:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Примеры:")
		fmt.Println("  duplicatelink /path/to/folder")
		fmt.Println("  duplicatelink -hash md5 -output results.json /path/to/folder")
		fmt.Println("  duplicatelink -mode clean -json results.json -target /path/to/storage")
		fmt.Println("  duplicatelink -mode restore -restore /path/to/storage/restore.json")
		fmt.Println("\nДокументация: README.md")
		fmt.Println("⚠️  ВНИМАНИЕ: см. DISCLAIMER.md перед использованием")
		os.Exit(0)
	}

	// Проверяем режим работы
	*mode = strings.ToLower(*mode)
	if *mode != "scan" && *mode != "clean" && *mode != "restore" {
		fmt.Printf("Ошибка: неподдерживаемый режим '%s'. Используйте scan, clean или restore\n", *mode)
		os.Exit(1)
	}

	// Режим восстановления
	if *mode == "restore" {
		if *restoreFile == "" {
			fmt.Println("Ошибка: для режима restore необходимо указать путь к JSON файлу через флаг -restore")
			os.Exit(1)
		}

		err := restore.RestoreFromBackup(*restoreFile)
		if err != nil {
			fmt.Printf("Ошибка при восстановлении: %v\n", err)
			os.Exit(1)
		}

		return
	}

	// Режим очистки
	if *mode == "clean" {
		if *jsonFile == "" {
			fmt.Println("Ошибка: для режима clean необходимо указать путь к JSON файлу через флаг -json")
			os.Exit(1)
		}

		if *targetDir == "" {
			fmt.Println("Ошибка: для режима clean необходимо указать папку назначения через флаг -target")
			os.Exit(1)
		}

		// Преобразуем targetDir в абсолютный путь
		absTargetDir, err := filepath.Abs(*targetDir)
		if err != nil {
			fmt.Printf("Ошибка: не удалось преобразовать путь targetDir в абсолютный: %v\n", err)
			os.Exit(1)
		}
		*targetDir = absTargetDir

		// Загружаем результаты из JSON
		result, err := scanner.LoadResultsFromJSON(*jsonFile)
		if err != nil {
			fmt.Printf("Ошибка при загрузке JSON: %v\n", err)
			os.Exit(1)
		}

		// Запрашиваем подтверждение
		if !cleaner.AskConfirmation() {
			fmt.Println("Операция отменена")
			os.Exit(0)
		}

		// Выполняем очистку
		err = cleaner.CleanDuplicates(result, *targetDir)
		if err != nil {
			fmt.Printf("Ошибка при очистке: %v\n", err)
			os.Exit(1)
		}

		return
	}

	// Режим сканирования
	// Получаем путь к папке из первого аргумента или используем текущую директорию
	folderPath := "."
	args := flag.Args()
	if len(args) > 0 {
		folderPath = args[0]
	}

	// Преобразуем в абсолютный путь
	folderPath, err := filepath.Abs(folderPath)
	if err != nil {
		fmt.Printf("Ошибка: не удалось преобразовать путь в абсолютный: %v\n", err)
		os.Exit(1)
	}

	// Проверяем, что папка существует
	info, err := os.Stat(folderPath)
	if err != nil {
		fmt.Printf("Ошибка: не удалось открыть папку %s: %v\n", folderPath, err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Printf("Ошибка: %s не является директорией\n", folderPath)
		os.Exit(1)
	}

	// Проверяем корректность алгоритма
	*hashAlg = strings.ToLower(*hashAlg)
	if *hashAlg != "sha1" && *hashAlg != "md5" {
		fmt.Printf("Ошибка: неподдерживаемый алгоритм '%s'. Используйте sha1 или md5\n", *hashAlg)
		os.Exit(1)
	}

	fmt.Printf("Поиск дубликатов в папке: %s\n", folderPath)
	fmt.Printf("Метод определения: %s\n", strings.ToUpper(*hashAlg))
	fmt.Printf("Количество воркеров: %d\n", *workers)
	if !*includeHidden {
		fmt.Println("Скрытые папки: пропускаются (используйте -include-hidden для их сканирования)")
	} else {
		fmt.Println("Скрытые папки: сканируются")
	}

	// Ищем дубликаты
	hashMap := scanner.FindDuplicates(folderPath, *hashAlg, *includeHidden, *workers)

	// Выводим результаты
	scanner.PrintDuplicates(hashMap, *hashAlg)

	// Сохраняем результаты в JSON
	err = scanner.SaveResultsToJSON(hashMap, folderPath, *hashAlg, *outputFile)
	if err != nil {
		fmt.Printf("Ошибка при сохранении результатов: %v\n", err)
		os.Exit(1)
	}

	// Сохраняем отчет сканирования в Markdown
	err = scanner.SaveScanReportMarkdown(hashMap, folderPath, *hashAlg, *scanReport)
	if err != nil {
		fmt.Printf("Ошибка при сохранении MD отчета: %v\n", err)
		os.Exit(1)
	}
}
