package scanner

import (
	"os"
	"path/filepath"
	"strings"
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
	hashMap := FindDuplicates(tmpDir, "sha1", false, 10)

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

func TestFindDuplicates_RespectsDupIgnore(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем структуру папок
	allowedDir := filepath.Join(tmpDir, "allowed")
	ignoredDir := filepath.Join(tmpDir, "ignored")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatalf("Не удалось создать allowedDir: %v", err)
	}
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatalf("Не удалось создать ignoredDir: %v", err)
	}

	// .dupignore: игнорировать папку ignored/
	ignoreContent := []byte("ignored/\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".dupignore"), ignoreContent, 0644); err != nil {
		t.Fatalf("Не удалось записать .dupignore: %v", err)
	}

	// Дубликаты в allowed
	a1 := filepath.Join(allowedDir, "file1.txt")
	a2 := filepath.Join(allowedDir, "file2.txt")
	// Такой же файл в ignored
	i1 := filepath.Join(ignoredDir, "file3.txt")
	content := []byte("same-data")
	if err := os.WriteFile(a1, content, 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}
	if err := os.WriteFile(a2, content, 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}
	if err := os.WriteFile(i1, content, 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}

	// Выполняем сканирование
	hashMap := FindDuplicates(tmpDir, "sha1", true, 10)
	if hashMap == nil {
		t.Fatalf("FindDuplicates вернул nil")
	}

	// Проверяем, что игнорируемый файл не попал в результаты
	for _, files := range hashMap {
		for _, f := range files {
			if strings.Contains(f.Path, "/ignored/") {
				t.Fatalf("Файл из игнорируемой папки попал в результаты: %s", f.Path)
			}
		}
	}

	// Ожидаем одну группу с двумя файлами из allowed
	found := false
	for _, files := range hashMap {
		if len(files) == 2 {
			paths := []string{files[0].Path, files[1].Path}
			if (paths[0] == a1 && paths[1] == a2) || (paths[0] == a2 && paths[1] == a1) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("Ожидалась группа дубликатов только из allowed, но она не найдена")
	}
}

func TestDupIgnore_NegationAllowsPhotosAndVideos(t *testing.T) {
	tmpDir := t.TempDir()

	// .dupignore: игнорировать всё, кроме фото и видео
	lines := []string{
		"*",
		"!*.jpg", "!*.jpeg", "!*.png", "!*.gif", "!*.bmp", "!*.webp", "!*.tiff", "!*.tif",
		"!*.mp4", "!*.mov", "!*.mkv", "!*.avi", "!*.wmv", "!*.flv", "!*.webm", "!*.m4v", "!*.mpg", "!*.mpeg",
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".dupignore"), []byte(content), 0644); err != nil {
		t.Fatalf("Не удалось записать .dupignore: %v", err)
	}

	files := map[string][]byte{
		filepath.Join(tmpDir, "photo.jpg"): []byte("img1"),
		filepath.Join(tmpDir, "video.mp4"): []byte("vid1"),
		filepath.Join(tmpDir, "doc.txt"):   []byte("text"),
	}
	for p, b := range files {
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatalf("Не удалось создать файл %s: %v", p, err)
		}
	}

	hashMap := FindDuplicates(tmpDir, "sha1", true, 10)
	// Собираем список включенных путей
	included := map[string]bool{}
	for _, arr := range hashMap {
		for _, fh := range arr {
			included[fh.Path] = true
		}
	}

	if !included[filepath.Join(tmpDir, "photo.jpg")] {
		t.Fatalf("photo.jpg должен быть включен")
	}
	if !included[filepath.Join(tmpDir, "video.mp4")] {
		t.Fatalf("video.mp4 должен быть включен")
	}
	if included[filepath.Join(tmpDir, "doc.txt")] {
		t.Fatalf("doc.txt должен быть исключен")
	}
}
