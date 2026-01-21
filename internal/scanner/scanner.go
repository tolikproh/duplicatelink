package scanner

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/tolikproh/duplicatelink/internal/models"
)

// FindDuplicates ищет дубликаты файлов по выбранному алгоритму хеширования
func FindDuplicates(rootPath string, hash string, includeHidden bool) map[string][]models.FileHash {
	hashMap := make(map[string][]models.FileHash)

	// Первый проход: считаем количество файлов
	fileCount := 0
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Пропускаем скрытые папки, если не включено сканирование скрытых
		if info.IsDir() && !includeHidden && isHidden(info.Name(), path, rootPath) {
			return filepath.SkipDir
		}

		if !info.IsDir() && (info.Mode()&os.ModeSymlink) == 0 {
			fileCount++
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка при обходе директории: %v\n", err)
		return nil
	}

	// Второй проход: обрабатываем файлы с прогресс-баром
	bar := progressbar.Default(int64(fileCount), fmt.Sprintf("Сканирование папки (%s)", strings.ToUpper(hash)))

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			bar.Add(1)
			return nil
		}

		// Пропускаем скрытые папки, если не включено сканирование скрытых
		if info.IsDir() && !includeHidden && isHidden(info.Name(), path, rootPath) {
			return filepath.SkipDir
		}

		// Пропускаем директории и символические ссылки
		if info.IsDir() || (info.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		// Вычисляем хеш файла
		hashValue, err := calculateHash(path, hash)
		if err != nil {
			bar.Add(1)
			return nil
		}

		// Добавляем файл в карту по хешу
		hashMap[hashValue] = append(hashMap[hashValue], models.FileHash{
			Path: path,
			Hash: hashValue,
			Size: info.Size(),
		})

		bar.Add(1)
		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка при обходе директории: %v\n", err)
		return nil
	}

	return hashMap
}

// isHidden проверяет, является ли файл или папка скрытой
func isHidden(name, path, rootPath string) bool {
	// Не считаем корневую папку скрытой
	if path == rootPath {
		return false
	}
	// Скрытые файлы и папки начинаются с точки
	return strings.HasPrefix(name, ".")
}

// calculateHash вычисляет хеш файла по выбранному алгоритму
func calculateHash(filePath string, hash string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash = strings.ToLower(hash)

	switch hash {
	case "sha1":
		hash := sha1.New()
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", hash.Sum(nil)), nil

	case "md5":
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", hash.Sum(nil)), nil

	default:
		return "", fmt.Errorf("неподдерживаемый алгоритм: %s", hash)
	}
}

// PrintDuplicates выводит найденные дубликаты
func PrintDuplicates(hashMap map[string][]models.FileHash, hash string) {
	duplicateCount := 0
	fileCount := 0

	// Сортируем ключи для консистентного вывода
	var hashes []string
	for hash := range hashMap {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)

	for _, hashValue := range hashes {
		files := hashMap[hashValue]
		fileCount += len(files)

		// Выводим только дубликаты (более одного файла с одинаковым хешом)
		if len(files) > 1 {
			duplicateCount++
			fmt.Printf("\n=== Дубликат #%d ===\n", duplicateCount)
			fmt.Printf("%s: %s\n", strings.ToUpper(hash), hashValue)
			fmt.Printf("Размер: %d байт\n", files[0].Size)
			fmt.Printf("Файлы:\n")

			for i, file := range files {
				fmt.Printf("  %d. %s\n", i+1, file.Path)
			}
		}
	}

	fmt.Printf("\n=== ИТОГО ===\n")
	fmt.Printf("Всего файлов обработано: %d\n", fileCount)
	fmt.Printf("Найдено групп дубликатов: %d\n", duplicateCount)
}

// SaveResultsToJSON сохраняет результаты сканирования в JSON файл
func SaveResultsToJSON(hashMap map[string][]models.FileHash, folderPath, hash, outputFile string) error {
	var duplicates []models.DuplicateGroup

	// Собираем только дубликаты
	for hashValue, files := range hashMap {
		if len(files) > 1 {
			duplicates = append(duplicates, models.DuplicateGroup{
				Hash:      hashValue,
				Size:      files[0].Size,
				Files:     files,
				Algorithm: strings.ToUpper(hash),
			})
		}
	}

	result := models.ScanResult{
		FolderPath: folderPath,
		Algorithm:  strings.ToUpper(hash),
		Duplicates: duplicates,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при создании JSON: %w", err)
	}

	err = os.WriteFile(outputFile, data, 0644)
	if err != nil {
		return fmt.Errorf("ошибка при записи файла: %w", err)
	}

	fmt.Printf("\nРезультаты сохранены в файл: %s\n", outputFile)
	return nil
}

// SaveScanReportMarkdown сохраняет отчет о сканировании в Markdown
func SaveScanReportMarkdown(hashMap map[string][]models.FileHash, folderPath, hash, reportFile string) error {
	var duplicates []models.DuplicateGroup
	totalFiles := 0

	for hashValue, files := range hashMap {
		totalFiles += len(files)
		if len(files) > 1 {
			duplicates = append(duplicates, models.DuplicateGroup{
				Hash:      hashValue,
				Size:      files[0].Size,
				Files:     files,
				Algorithm: strings.ToUpper(hash),
			})
		}
	}

	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].Hash < duplicates[j].Hash })

	var b strings.Builder
	b.WriteString("# Отчет о сканировании дубликатов\n\n")
	b.WriteString(fmt.Sprintf("**Папка:** %s\n\n", folderPath))
	b.WriteString(fmt.Sprintf("**Алгоритм:** %s\n\n", strings.ToUpper(hash)))
	b.WriteString(fmt.Sprintf("**Всего файлов проверено:** %d\n\n", totalFiles))
	b.WriteString(fmt.Sprintf("**Групп дубликатов:** %d\n\n", len(duplicates)))
	b.WriteString("---\n\n")

	for idx, group := range duplicates {
		b.WriteString(fmt.Sprintf("## Группа %d\n\n", idx+1))
		b.WriteString(fmt.Sprintf("**Хеш (%s):** `%s`\n\n", strings.ToUpper(hash), group.Hash))
		b.WriteString(fmt.Sprintf("**Размер:** %d байт\n\n", group.Size))
		b.WriteString("**Файлы:**\n\n")
		for _, f := range group.Files {
			b.WriteString(fmt.Sprintf("- [%s](file://%s)\n", f.Path, f.Path))
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(reportFile, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("ошибка при сохранении MD отчета: %w", err)
	}

	fmt.Printf("Отчет сканирования сохранен: %s\n", reportFile)
	return nil
}

// LoadResultsFromJSON загружает результаты сканирования из JSON файла
func LoadResultsFromJSON(inputFile string) (*models.ScanResult, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	var result models.ScanResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return &result, nil
}
