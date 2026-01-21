package dupignore

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// entry представляет одну строку правила: игнор или разрешение (!)
type entry struct {
	pattern string
	allow   bool // true, если правило разрешающее (начинается с '!')
}

// Rules хранит список правил из .dupignore в порядке следования
type Rules struct {
	entries []entry
}

// Load загружает файл .dupignore из корня,
// парсит строки и возвращает правила. Поддерживаются базовые паттерны по аналогии .gitignore:
// - Директории: "ignored/" — игнорировать всё внутри
// - Глоб: "*.bak", "temp/*" — проверяется относительно пути от корня
// - Комментарии: строки начинающиеся с '#'
func Load(rootPath string) *Rules {
	pathNew := filepath.Join(rootPath, ".dupignore")

	var f *os.File
	var err error
	if _, statErr := os.Stat(pathNew); statErr == nil {
		f, err = os.Open(pathNew)
	} else {
		return nil
	}
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []entry
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Нормализуем слэши и удаляем ведущие './'
		allow := false
		if strings.HasPrefix(line, "!") {
			allow = true
			line = strings.TrimPrefix(line, "!")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
		}

		line = filepath.ToSlash(line)
		line = strings.TrimPrefix(line, "./")
		entries = append(entries, entry{pattern: line, allow: allow})
	}
	if len(entries) == 0 {
		return nil
	}
	return &Rules{entries: entries}
}

// Matches проверяет, соответствует ли путь хотя бы одному паттерну из правил.
// Путь приводится к относительному от rootPath и нормализуется к слэшам '/'.
func Matches(r *Rules, rootPath, absPath string, isDir bool) bool {
	if r == nil {
		return false
	}
	rel := absPath
	if strings.HasPrefix(absPath, rootPath) {
		if v, err := filepath.Rel(rootPath, absPath); err == nil {
			rel = v
		}
	}
	rel = filepath.ToSlash(rel)
	base := path.Base(rel)
	ignored := false
	for _, e := range r.entries {
		p := e.pattern
		// Директория: "dir/" — префикс
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(rel, p) {
				ignored = !e.allow
			}
			continue
		}
		// Если это директория, правила без суффикса '/' к ней не применяем
		if isDir {
			continue
		}

		// Сначала проверяем глоб по полному относительному пути
		if ok, _ := path.Match(p, rel); ok {
			ignored = !e.allow
			continue
		}
		// Затем по имени файла
		if ok, _ := path.Match(p, base); ok {
			ignored = !e.allow
			continue
		}
		// Простая проверка префикса (для паттернов без спецсимволов)
		if !strings.ContainsAny(p, "*?[]") {
			if strings.HasPrefix(rel, p) {
				ignored = !e.allow
				continue
			}
		}
	}
	return ignored
}
