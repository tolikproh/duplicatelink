package dupignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNegationAllowsExtensions(t *testing.T) {
	tmp := t.TempDir()
	content := "*\n!*.jpg\n"
	if err := os.WriteFile(filepath.Join(tmp, ".dupignore"), []byte(content), 0644); err != nil {
		t.Fatalf("write dupignore: %v", err)
	}
	r := Load(tmp)
	if r == nil {
		t.Fatalf("rules should load")
	}
	p := filepath.Join(tmp, "photo.jpg")
	if Matches(r, tmp, p, false) {
		t.Fatalf("photo.jpg should NOT be ignored due to negation")
	}
	q := filepath.Join(tmp, "doc.txt")
	if !Matches(r, tmp, q, false) {
		t.Fatalf("doc.txt should be ignored by '*' rule")
	}
}
