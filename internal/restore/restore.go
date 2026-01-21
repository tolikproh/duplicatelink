package restore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tolikproh/duplicatelink/internal/models"
)

// RestoreFromBackup восстанавливает исходное состояние
func RestoreFromBackup(restorePath string) error {
	fmt.Println("\n=== РЕЖИМ ВОССТАНОВЛЕНИЯ ===")

	// Загружаем данные восстановления
	data, err := os.ReadFile(restorePath)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла восстановления: %w", err)
	}

	var restoreData models.RestoreData
	if err := json.Unmarshal(data, &restoreData); err != nil {
		return fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	fmt.Printf("Дата бэкапа: %s\n", restoreData.Timestamp)
	fmt.Printf("Папка назначения: %s\n", restoreData.TargetDir)
	fmt.Printf("Действий для восстановления: %d\n", len(restoreData.Actions))

	// Запрашиваем подтверждение
	fmt.Print("\n⚠️  Вы собираетесь восстановить исходное состояние.\n")
	fmt.Print("Симлинки будут удалены, оригинальные файлы восстановлены. Продолжить? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("ошибка при чтении ответа: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Операция отменена")
		return nil
	}

	// Восстанавливаем файлы
	for idx, action := range restoreData.Actions {
		fmt.Printf("\nВосстановление %d/%d...\n", idx+1, len(restoreData.Actions))

		if action.IsSymlink {
			// Удаляем симлинк
			if err := os.Remove(action.SymlinkPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("⚠️  Ошибка при удалении симлинка %s: %v\n", action.SymlinkPath, err)
				continue
			}

			// Копируем оригинальный файл обратно
			if err := copyFile(action.MovedTo, action.OriginalPath); err != nil {
				fmt.Printf("⚠️  Ошибка при восстановлении %s: %v\n", action.OriginalPath, err)
				continue
			}

			fmt.Printf("✓ Восстановлен: %s\n", action.OriginalPath)
		}
	}

	fmt.Println("\n✅ Восстановление завершено!")
	fmt.Printf("⚠️  Примечание: файлы в папке %s остались без изменений.\n", restoreData.TargetDir)
	fmt.Println("Вы можете удалить эту папку вручную, если больше не нужна.")

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
