package cleaner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// Создаем папку назначения, если не существует
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("ошибка при создании папки назначения: %w", err)
	}

	var restoreActions []models.RestoreAction
	var errorList []models.ErrorInfo
	var mdReport strings.Builder

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

		// Выбираем первый файл для сохранения
		masterFile := group.Files[0]
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

		// Обрабатываем дубликаты
		for _, duplicate := range group.Files {
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

			restoreActions = append(restoreActions, models.RestoreAction{
				OriginalPath: duplicate.Path,
				MovedTo:      destPath,
				SymlinkPath:  duplicate.Path,
				IsSymlink:    true,
			})

			mdReport.WriteString(fmt.Sprintf("- [%s](file://%s)\n", duplicate.Path, duplicate.Path))
		}
		mdReport.WriteString("\n")
	}

	// Сохраняем MD отчет
	reportPath := filepath.Join(targetDir, "cleanup_report.md")
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
