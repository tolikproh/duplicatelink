package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tolikproh/duplicatelink/internal/models"
)

func TestCalculateHash(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")

	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Не удалось создать тестовый файл: %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		expected string
		wantErr  bool
	}{
		{
			name:     "SHA1 hash",
			hash:     "sha1",
			expected: "0a0a9f2a6772942557ab5355d76af442f8f65e01",
			wantErr:  false,
		},
		{
			name:     "MD5 hash",
			hash:     "md5",
			expected: "65a8e27d8879283831b664bd8b7f0ad4",
			wantErr:  false,
		},
		{
			name:     "Unsupported hash",
			hash:     "sha256",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculateHash(testFile, tt.hash)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Ожидалась ошибка, но её не было")
				}
				return
			}

			if err != nil {
				t.Errorf("Неожиданная ошибка: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("calculateHash() = %v, ожидалось %v", result, tt.expected)
			}
		})
	}
}

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		path     string
		rootPath string
		expected bool
	}{
		{
			name:     "Hidden file",
			fileName: ".hidden",
			path:     "/home/user/.hidden",
			rootPath: "/home/user",
			expected: true,
		},
		{
			name:     "Normal file",
			fileName: "file.txt",
			path:     "/home/user/file.txt",
			rootPath: "/home/user",
			expected: false,
		},
		{
			name:     "Root path",
			fileName: "user",
			path:     "/home/user",
			rootPath: "/home/user",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHidden(tt.fileName, tt.path, tt.rootPath)
			if result != tt.expected {
				t.Errorf("isHidden() = %v, ожидалось %v", result, tt.expected)
			}
		})
	}
}

func TestSaveAndLoadResultsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "results.json")

	hashMap := map[string][]models.FileHash{
		"abc123": {
			{Path: "/path/file1.txt", Hash: "abc123", Size: 100},
			{Path: "/path/file2.txt", Hash: "abc123", Size: 100},
		},
	}

	err := SaveResultsToJSON(hashMap, "/test/path", "sha1", outputFile)
	if err != nil {
		t.Fatalf("SaveResultsToJSON вернул ошибку: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Файл результатов не был создан")
	}

	result, err := LoadResultsFromJSON(outputFile)
	if err != nil {
		t.Fatalf("LoadResultsFromJSON вернул ошибку: %v", err)
	}

	if result.FolderPath != "/test/path" {
		t.Errorf("FolderPath = %v, ожидалось /test/path", result.FolderPath)
	}

	if result.Algorithm != "SHA1" {
		t.Errorf("Algorithm = %v, ожидалось SHA1", result.Algorithm)
	}

	if len(result.Duplicates) != 1 {
		t.Errorf("Ожидалось 1 группа дубликатов, получено %d", len(result.Duplicates))
	}
}

func TestFindDuplicates_SkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем обычный файл и симлинк на него
	original := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(original, []byte("data"), 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}

	symlink := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(original, symlink); err != nil {
		t.Fatalf("Не удалось создать симлинк: %v", err)
	}

	// Выполняем сканирование
	hashMap := FindDuplicates(tmpDir, "sha1", false)

	if hashMap == nil {
		t.Fatalf("FindDuplicates вернул nil")
	}

	if len(hashMap) != 1 {
		t.Fatalf("Ожидался один уникальный хеш, получено %d", len(hashMap))
	}

	for _, files := range hashMap {
		if len(files) != 1 {
			t.Fatalf("Симлинк должен быть пропущен, но получено %d файлов", len(files))
		}

		if files[0].Path != original {
			t.Fatalf("Ожидался только исходный файл, получено %s", files[0].Path)
		}
	}
}
