package cleaner

import (
"os"
"path/filepath"
"testing"
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
