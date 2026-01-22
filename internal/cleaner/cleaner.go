package cleaner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	dupignore "github.com/tolikproh/duplicatelink/internal/dupignore"
	"github.com/tolikproh/duplicatelink/internal/models"
)

// AskConfirmation запрашивает подтверждение у пользователя
func AskConfirmation() bool {
	fmt.Print("\n⚠️  ВНИМАНИЕ! Вы собираетесь удалить дублирующиеся файлы.\n")
	fmt.Print("Это действие необратимо. Продолжить? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
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

// getFileCategory определяет категорию файла по расширению
func getFileCategory(filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	categories := map[string][]string{
		"image":    {"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg", "ico", "tiff", "tif"},
		"video":    {"mp4", "avi", "mkv", "mov", "wmv", "flv", "webm", "m4v", "mpg", "mpeg"},
		"audio":    {"mp3", "wav", "flac", "aac", "ogg", "wma", "m4a", "opus"},
		"document": {"pdf", "doc", "docx", "txt", "rtf", "odt", "tex", "md"},
		"archive":  {"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "iso"},
		"code":     {"go", "py", "js", "java", "cpp", "c", "h", "rs", "rb", "php", "html", "css"},
		"data":     {"json", "xml", "yaml", "yml", "csv", "sql", "db"},
	}

	for category, extensions := range categories {
		for _, e := range extensions {
			if ext == e {
				return category
			}
		}
	}

	return "other"
}

// CleanDuplicates выполняет автоматическую очистку дубликатов
func CleanDuplicates(result *models.ScanResult, targetDir string) error {
	fmt.Println("\n=== РЕЖИМ ОЧИСТКИ ===")
	fmt.Printf("Папка источник: %s\n", result.FolderPath)
	fmt.Printf("Папка назначения: %s\n", targetDir)
	fmt.Printf("Алгоритм: %s\n", result.Algorithm)
	fmt.Printf("Групп дубликатов: %d\n", len(result.Duplicates))

	// Загружаем правила игнора из корня исходной папки (.dupignore)
	ign := dupignore.Load(result.FolderPath)

	// Создаем папку назначения, если не существует
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("ошибка при создании папки назначения: %w", err)
	}

	var restoreActions []models.RestoreAction
	var errorList []models.ErrorInfo
	var mdReport strings.Builder
	var totalSpaceFreed int64 = 0
	var processedGroups int = 0
	var totalSymlinksCreated int = 0

	mdReport.WriteString("# Отчет об очистке дубликатов\n\n")
	mdReport.WriteString(fmt.Sprintf("**Дата:** %s\n\n", filepath.Base(targetDir)))
	mdReport.WriteString(fmt.Sprintf("**Исходная папка:** %s\n\n", result.FolderPath))
	mdReport.WriteString(fmt.Sprintf("**Алгоритм:** %s\n\n", result.Algorithm))
	mdReport.WriteString(fmt.Sprintf("**Групп дубликатов:** %d\n\n", len(result.Duplicates)))
	mdReport.WriteString("---\n\n")

	for idx, group := range result.Duplicates {
		if len(group.Files) < 2 {
			continue
		}

		fmt.Printf("\nОбработка группы %d/%d...\n", idx+1, len(result.Duplicates))

		// Отфильтровываем файлы по .dupignore
		var active []models.FileHash
		for _, f := range group.Files {
			if dupignore.Matches(ign, result.FolderPath, f.Path, false) {
				continue
			}
			active = append(active, f)
		}
		if len(active) < 1 {
			// Все файлы в группе игнорируются
			continue
		}

		// Выбираем первый неигнорируемый файл для сохранения
		masterFile := active[0]
		category := getFileCategory(masterFile.Path)

		// Создаем папку для категории
		categoryDir := filepath.Join(targetDir, category)
		if err := os.MkdirAll(categoryDir, 0755); err != nil {
			return fmt.Errorf("ошибка при создании папки категории: %w", err)
		}

		// Определяем путь назначения
		filename := filepath.Base(masterFile.Path)
		destPath := filepath.Join(categoryDir, filename)

		// Если файл с таким именем уже существует, добавляем префикс
		counter := 1
		for {
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
			ext := filepath.Ext(filename)
			nameWithoutExt := strings.TrimSuffix(filename, ext)
			destPath = filepath.Join(categoryDir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
			counter++
		}

		// Преобразуем destPath в абсолютный путь (на случай если filepath.Join вернул относительный)
		absDestPath, absErr := filepath.Abs(destPath)
		if absErr != nil {
			fmt.Printf("⚠️  Ошибка при получении абсолютного пути для %s: %v\n", destPath, absErr)
			errorList = append(errorList, models.ErrorInfo{
				FilePath:  masterFile.Path,
				Operation: "abs_path",
				Error:     absErr.Error(),
				Timestamp: fmt.Sprintf("%d", idx),
				ErrorType: "abs_path_failed",
			})
			continue
		}
		destPath = absDestPath

		// Копируем мастер-файл
		if err := copyFile(masterFile.Path, destPath); err != nil {
			fmt.Printf("⚠️  Ошибка при копировании %s: %v\n", masterFile.Path, err)
			errorList = append(errorList, models.ErrorInfo{
				FilePath:  masterFile.Path,
				Operation: "copy",
				Error:     err.Error(),
				Timestamp: fmt.Sprintf("%d", idx),
				ErrorType: "copy_failed",
			})
			continue
		}

		fmt.Printf("✓ Скопирован: %s -> %s\n", masterFile.Path, destPath)

		restoreActions = append(restoreActions, models.RestoreAction{
			OriginalPath: masterFile.Path,
			MovedTo:      destPath,
			IsSymlink:    false,
		})

		// Добавляем в отчет
		mdReport.WriteString(fmt.Sprintf("## Группа %d\n\n", idx+1))
		mdReport.WriteString(fmt.Sprintf("**Хеш (%s):** `%s`\n\n", strings.ToUpper(result.Algorithm), group.Hash))
		mdReport.WriteString(fmt.Sprintf("**Размер:** %d байт\n\n", group.Size))
		mdReport.WriteString(fmt.Sprintf("**Категория:** %s\n\n", category))
		mdReport.WriteString(fmt.Sprintf("**Основной файл:** [%s](file://%s)\n\n", filename, destPath))
		mdReport.WriteString("**Дубликаты (заменены симлинками):**\n\n")

		// Обрабатываем дубликаты (только неигнорируемые)
		var groupSpaceFreed int64 = 0
		var groupSymlinksCreated int = 0
		for _, duplicate := range active {
			// Удаляем дубликат и создаем симлинк
			removeErr := os.Remove(duplicate.Path)
			if removeErr != nil {
				fmt.Printf("⚠️  Ошибка при удалении %s: %v\n", duplicate.Path, removeErr)
				errorList = append(errorList, models.ErrorInfo{
					FilePath:  duplicate.Path,
					Operation: "remove",
					Error:     removeErr.Error(),
					Timestamp: fmt.Sprintf("%d", idx),
					ErrorType: "remove_failed",
				})
				mdReport.WriteString(fmt.Sprintf("- ⚠️ [%s](file://%s) - **ОШИБКА при удалении**: %s\n", duplicate.Path, duplicate.Path, removeErr.Error()))
				continue
			}

			// Создаем симлинк
			symlinkErr := os.Symlink(destPath, duplicate.Path)
			if symlinkErr != nil {
				fmt.Printf("⚠️  Ошибка при создании симлинка %s: %v\n", duplicate.Path, symlinkErr)
				errorList = append(errorList, models.ErrorInfo{
					FilePath:  duplicate.Path,
					Operation: "symlink",
					Error:     symlinkErr.Error(),
					Timestamp: fmt.Sprintf("%d", idx),
					ErrorType: "symlink_failed",
				})
				mdReport.WriteString(fmt.Sprintf("- ⚠️ [%s](file://%s) - **ОШИБКА при создании симлинка**: %s\n", duplicate.Path, duplicate.Path, symlinkErr.Error()))
				continue
			}

			fmt.Printf("✓ Создан симлинк: %s -> %s\n", duplicate.Path, destPath)

			// Подсчитываем освобожденное место (размер файла, так как симлинк почти не занимает места)
			groupSpaceFreed += duplicate.Size
			groupSymlinksCreated++

			restoreActions = append(restoreActions, models.RestoreAction{
				OriginalPath: duplicate.Path,
				MovedTo:      destPath,
				SymlinkPath:  duplicate.Path,
				IsSymlink:    true,
			})

			mdReport.WriteString(fmt.Sprintf("- [%s](file://%s)\n", duplicate.Path, duplicate.Path))
		}

		// Добавляем информацию об освобожденном месте в этой группе
		if groupSymlinksCreated > 0 {
			mdReport.WriteString(fmt.Sprintf("\n**💾 Освобождено в этой группе:** %s (%d байт)\n", formatSize(groupSpaceFreed), groupSpaceFreed))
			mdReport.WriteString(fmt.Sprintf("**Создано симлинков:** %d\n", groupSymlinksCreated))
			totalSpaceFreed += groupSpaceFreed
			totalSymlinksCreated += groupSymlinksCreated
			processedGroups++
		}
		mdReport.WriteString("\n")
	}

	// Сохраняем MD отчет
	reportPath := filepath.Join(targetDir, "cleanup_report.md")

	// Добавляем итоговую статистику
	mdReport.WriteString("\n---\n\n")
	mdReport.WriteString("# 📋 Итоговая статистика\n\n")
	mdReport.WriteString(fmt.Sprintf("**Обработано групп:** %d\n\n", processedGroups))
	mdReport.WriteString(fmt.Sprintf("**Создано симлинков:** %d\n\n", totalSymlinksCreated))
	mdReport.WriteString(fmt.Sprintf("**💾 Всего освобождено места:** %s (%d байт)\n\n", formatSize(totalSpaceFreed), totalSpaceFreed))
	mdReport.WriteString("\n")

	// Добавляем раздел с ошибками, если они были
	if len(errorList) > 0 {
		mdReport.WriteString("\n---\n\n")
		mdReport.WriteString("# ⚠️ Ошибки при обработке\n\n")
		mdReport.WriteString(fmt.Sprintf("**Всего ошибок:** %d\n\n", len(errorList)))
		mdReport.WriteString("| Файл | Операция | Тип ошибки | Описание |\n")
		mdReport.WriteString("|------|----------|-----------|----------|\n")

		for _, errInfo := range errorList {
			mdReport.WriteString(fmt.Sprintf("| [%s](file://%s) | %s | %s | %s |\n",
				filepath.Base(errInfo.FilePath),
				errInfo.FilePath,
				errInfo.Operation,
				errInfo.ErrorType,
				errInfo.Error))
		}
		mdReport.WriteString("\n")
		fmt.Printf("\n⚠️  При обработке произошло %d ошибок. См. раздел \"Ошибки при обработке\" в отчёте.\n", len(errorList))
	} else {
		mdReport.WriteString("\n---\n\n✅ **Все файлы обработаны успешно без ошибок!**\n\n")
		fmt.Println("\n✅ Все файлы обработаны успешно без ошибок!")
	}

	// Выводим итоговую статистику в консоль
	fmt.Printf("\n=== ИТОГОВАЯ СТАТИСТИКА ===\n")
	fmt.Printf("Обработано групп: %d\n", processedGroups)
	fmt.Printf("Создано симлинков: %d\n", totalSymlinksCreated)
	fmt.Printf("💾 Всего освобождено места: %s (%d байт)\n", formatSize(totalSpaceFreed), totalSpaceFreed)

	// Сохраняем MD отчет
	if err := os.WriteFile(reportPath, []byte(mdReport.String()), 0644); err != nil {
		return fmt.Errorf("ошибка при сохранении отчета: %w", err)
	}
	fmt.Printf("\n✓ Отчет сохранен: %s\n", reportPath)

	// Сохраняем данные для восстановления
	restoreData := models.RestoreData{
		Timestamp:  filepath.Base(targetDir),
		SourceJSON: "",
		TargetDir:  targetDir,
		Actions:    restoreActions,
		Errors:     errorList,
		ReportFile: reportPath,
	}

	restorePath := filepath.Join(targetDir, "restore.json")
	restoreJSON, err := json.MarshalIndent(restoreData, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при создании JSON восстановления: %w", err)
	}

	if err := os.WriteFile(restorePath, restoreJSON, 0644); err != nil {
		return fmt.Errorf("ошибка при сохранении JSON восстановления: %w", err)
	}
	fmt.Printf("✓ Файл восстановления сохранен: %s\n", restorePath)

	fmt.Println("\n✅ Очистка завершена успешно!")

	// Если есть ошибки, создаем отдельный JSON файл только для них
	if len(errorList) > 0 {
		errorOnlyData := models.RestoreData{
			Timestamp:  filepath.Base(targetDir),
			SourceJSON: "",
			TargetDir:  targetDir,
			Actions:    []models.RestoreAction{}, // Пустой список действий
			Errors:     errorList,
			ReportFile: reportPath,
		}

		errorPath := filepath.Join(targetDir, "restore_errors_only.json")
		errorJSON, err := json.MarshalIndent(errorOnlyData, "", "  ")
		if err != nil {
			return fmt.Errorf("ошибка при создании JSON ошибок: %w", err)
		}

		if err := os.WriteFile(errorPath, errorJSON, 0644); err != nil {
			return fmt.Errorf("ошибка при сохранении JSON ошибок: %w", err)
		}
		fmt.Printf("✓ Файл ошибок сохранен: %s\n", errorPath)
	}

	return nil
}

// copyFile копирует файл
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Копируем права доступа
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
