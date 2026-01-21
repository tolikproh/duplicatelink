package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tolikproh/duplicatelink/internal/models"
)

func TestGetFileCategory(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"JPEG image", "photo.jpg", "image"},
		{"PNG image", "screenshot.png", "image"},
		{"MP4 video", "movie.mp4", "video"},
		{"MP3 audio", "song.mp3", "audio"},
		{"PDF document", "report.pdf", "document"},
		{"ZIP archive", "backup.zip", "archive"},
		{"Go code", "main.go", "code"},
		{"JSON data", "config.json", "data"},
		{"Unknown extension", "file.xyz", "other"},
		{"No extension", "README", "other"},
		{"Case insensitive", "PHOTO.JPG", "image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFileCategory(tt.filename)
			if result != tt.expected {
				t.Errorf("getFileCategory(%q) = %v, ожидалось %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	err := os.WriteFile(srcPath, content, 0644)
	if err != nil {
		t.Fatalf("Не удалось создать исходный файл: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "destination.txt")
	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile вернул ошибку: %v", err)
	}

	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatal("Файл назначения не был создан")
	}

	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Не удалось прочитать файл назначения: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("Содержимое не совпадает")
	}
}

func TestCleanDuplicates_RespectsDupIgnore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Симлинки могут требовать привилегий на Windows")
	}
	tmpDir := t.TempDir()

	allowedDir := filepath.Join(tmpDir, "allowed")
	ignoredDir := filepath.Join(tmpDir, "ignored")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatalf("Не удалось создать allowedDir: %v", err)
	}
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatalf("Не удалось создать ignoredDir: %v", err)
	}

	// Игнорируем папку ignored/
	if err := os.WriteFile(filepath.Join(tmpDir, ".dupignore"), []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("Не удалось записать .dupignore: %v", err)
	}

	// Создаем дубликаты
	allowed := filepath.Join(allowedDir, "dup.txt")
	ignored := filepath.Join(ignoredDir, "dup.txt")
	content := []byte("same-data-clean")
	if err := os.WriteFile(allowed, content, 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}
	if err := os.WriteFile(ignored, content, 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}

	// Формируем результат сканирования: порядок — игнорируемый первым
	res := &models.ScanResult{
		FolderPath: tmpDir,
		Algorithm:  "SHA1",
		Duplicates: []models.DuplicateGroup{
			{
				Hash: "dummy",
				Size: int64(len(content)),
				Files: []models.FileHash{
					{Path: ignored, Hash: "dummy", Size: int64(len(content))},
					{Path: allowed, Hash: "dummy", Size: int64(len(content))},
				},
				Algorithm: "SHA1",
			},
		},
	}

	target := filepath.Join(tmpDir, "target")
	if err := CleanDuplicates(res, target); err != nil {
		t.Fatalf("CleanDuplicates вернул ошибку: %v", err)
	}

	// Проверяем, что файл в ignored остался обычным файлом
	stIgnored, err := os.Lstat(ignored)
	if err != nil {
		t.Fatalf("Не удалось Lstat ignored: %v", err)
	}
	if (stIgnored.Mode() & os.ModeSymlink) != 0 {
		t.Fatalf("Игнорируемый файл не должен быть симлинком")
	}

	// Проверяем, что файл в allowed стал симлинком
	stAllowed, err := os.Lstat(allowed)
	if err != nil {
		t.Fatalf("Не удалось Lstat allowed: %v", err)
	}
	if (stAllowed.Mode() & os.ModeSymlink) == 0 {
		t.Fatalf("Неигнорируемый файл должен быть заменён симлинком")
	}
}
