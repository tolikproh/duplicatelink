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
	"sync"

	"github.com/schollz/progressbar/v3"
	dupignore "github.com/tolikproh/duplicatelink/internal/dupignore"
	"github.com/tolikproh/duplicatelink/internal/models"
)

// SafeHashMap потокобезопасная структура для хранения результатов хеширования
type SafeHashMap struct {
	mu   sync.Mutex
	data map[string][]models.FileHash
}

// NewSafeHashMap создает новую потокобезопасную карту с начальной емкостью
func NewSafeHashMap(capacity int) *SafeHashMap {
	return &SafeHashMap{
		data: make(map[string][]models.FileHash, capacity),
	}
}

// Add добавляет файл в карту по его хешу
func (shm *SafeHashMap) Add(hash string, fileHash models.FileHash) {
	shm.mu.Lock()
	defer shm.mu.Unlock()
	shm.data[hash] = append(shm.data[hash], fileHash)
}

// ToMap возвращает обычную карту (для дальнейшей работы)
func (shm *SafeHashMap) ToMap() map[string][]models.FileHash {
	shm.mu.Lock()
	defer shm.mu.Unlock()
	return shm.data
}

// fileJob описывает задачу хеширования файла
type fileJob struct {
	path string
	size int64
}

// FindDuplicates ищет дубликаты файлов по выбранному алгоритму хеширования с использованием воркеров
func FindDuplicates(rootPath string, hash string, includeHidden bool, workers int) map[string][]models.FileHash {
	// Загружаем правила игнора, если есть .dupignore в корне
	ign := dupignore.Load(rootPath)

	// Этап 1: Собираем список всех файлов для сканирования
	fmt.Println("\n=== Этап 1: Сбор списка файлов ===")

	var filesToScan []fileJob
	var fileCount int64 = 0

	// Первый проход для подсчета файлов
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Пропускаем скрытые папки, если не включено сканирование скрытых
		if info.IsDir() && !includeHidden && isHidden(info.Name(), path, rootPath) {
			return filepath.SkipDir
		}

		// Пропускаем по правилам .dupignore
		if dupignore.Matches(ign, rootPath, path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

	// Прогресс-бар для сбора списка файлов
	collectBar := progressbar.Default(fileCount, "Сбор списка файлов")

	// Второй проход: собираем файлы в список
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			collectBar.Add(1)
			return nil
		}

		// Пропускаем скрытые папки, если не включено сканирование скрытых
		if info.IsDir() && !includeHidden && isHidden(info.Name(), path, rootPath) {
			return filepath.SkipDir
		}

		// Пропускаем по правилам .dupignore
		if dupignore.Matches(ign, rootPath, path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			collectBar.Add(1)
			return nil
		}

		// Пропускаем директории и символические ссылки
		if info.IsDir() || (info.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		// Преобразуем путь в абсолютный
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path // Если не удалось преобразовать, используем исходный путь
		}

		filesToScan = append(filesToScan, fileJob{
			path: absPath,
			size: info.Size(),
		})

		collectBar.Add(1)
		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка при обходе директории: %v\n", err)
		return nil
	}

	collectBar.Finish()
	fmt.Printf("Найдено файлов для сканирования: %d\n", len(filesToScan))

	// Этап 2: Хеширование файлов с помощью воркеров
	fmt.Println("\n=== Этап 2: Хеширование файлов ===")
	fmt.Printf("Количество воркеров: %d\n", workers)

	// Создаем потокобезопасную карту с начальной емкостью
	hashMap := NewSafeHashMap(len(filesToScan))

	// Создаем каналы для воркеров
	jobs := make(chan fileJob, len(filesToScan))
	var wg sync.WaitGroup

	// Прогресс-бар для хеширования
	hashBar := progressbar.Default(int64(len(filesToScan)), fmt.Sprintf("Хеширование (%s)", strings.ToUpper(hash)))

	// Запускаем воркеры
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Вычисляем хеш файла
				hashValue, err := calculateHash(job.path, hash)
				if err != nil {
					hashBar.Add(1)
					continue
				}

				// Добавляем файл в потокобезопасную карту
				hashMap.Add(hashValue, models.FileHash{
					Path: job.path,
					Hash: hashValue,
					Size: job.size,
				})

				hashBar.Add(1)
			}
		}()
	}

	// Отправляем задачи воркерам
	for _, job := range filesToScan {
		jobs <- job
	}
	close(jobs)

	// Ждем завершения всех воркеров
	wg.Wait()
	hashBar.Finish()

	fmt.Println("\n=== Сканирование завершено ===")
	return hashMap.ToMap()
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

// formatSize форматирует размер в байтах в читаемый вид
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d байт", size)
	}
}

// PrintDuplicates выводит найденные дубликаты
func PrintDuplicates(hashMap map[string][]models.FileHash, hash string) {
	duplicateCount := 0
	fileCount := 0
	var potentialSavings int64 = 0

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
			// Расчет освобождаемого места: (количество копий - 1) * размер
			groupSavings := int64(len(files)-1) * files[0].Size
			potentialSavings += groupSavings
			fmt.Printf("Освободится при очистке: %s\n", formatSize(groupSavings))
			fmt.Printf("Файлы:\n")

			for i, file := range files {
				fmt.Printf("  %d. %s\n", i+1, file.Path)
			}
		}
	}

	fmt.Printf("\n=== ИТОГО ===\n")
	fmt.Printf("Всего файлов обработано: %d\n", fileCount)
	fmt.Printf("Найдено групп дубликатов: %d\n", duplicateCount)
	fmt.Printf("Потенциально освободится места: %s (%d байт)\n", formatSize(potentialSavings), potentialSavings)
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
	var potentialSavings int64 = 0

	for hashValue, files := range hashMap {
		totalFiles += len(files)
		if len(files) > 1 {
			duplicates = append(duplicates, models.DuplicateGroup{
				Hash:      hashValue,
				Size:      files[0].Size,
				Files:     files,
				Algorithm: strings.ToUpper(hash),
			})
			// Расчет освобождаемого места
			potentialSavings += int64(len(files)-1) * files[0].Size
		}
	}

	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].Hash < duplicates[j].Hash })

	var b strings.Builder
	b.WriteString("# Отчет о сканировании дубликатов\n\n")
	b.WriteString(fmt.Sprintf("**Папка:** %s\n\n", folderPath))
	b.WriteString(fmt.Sprintf("**Алгоритм:** %s\n\n", strings.ToUpper(hash)))
	b.WriteString(fmt.Sprintf("**Всего файлов проверено:** %d\n\n", totalFiles))
	b.WriteString(fmt.Sprintf("**Групп дубликатов:** %d\n\n", len(duplicates)))
	b.WriteString(fmt.Sprintf("**💾 Потенциально освободится места:** %s (%d байт)\n\n", formatSize(potentialSavings), potentialSavings))
	b.WriteString("---\n\n")

	for idx, group := range duplicates {
		b.WriteString(fmt.Sprintf("## Группа %d\n\n", idx+1))
		b.WriteString(fmt.Sprintf("**Хеш (%s):** `%s`\n\n", strings.ToUpper(hash), group.Hash))
		b.WriteString(fmt.Sprintf("**Размер файла:** %d байт\n\n", group.Size))
		groupSavings := int64(len(group.Files)-1) * group.Size
		b.WriteString(fmt.Sprintf("**💾 Освободится:** %s (%d байт)\n\n", formatSize(groupSavings), groupSavings))
		b.WriteString(fmt.Sprintf("**Количество копий:** %d\n\n", len(group.Files)))
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
