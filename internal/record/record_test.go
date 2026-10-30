package record

import "testing"

func TestSlugOf(t *testing.T) {
	if got, ok := SlugOf("2026-10-30-fix-x.md"); !ok || got != "2026-10-30-fix-x" {
		t.Fatalf("SlugOf = %q, %v", got, ok)
	}
	for _, name := range []string{"README.md", "2026-10-30-fix-x.txt", ".md", "2026-10-30-fix-x", "2026-10-30.md", "2026-10-30-.md", "notes.md"} {
		if got, ok := SlugOf(name); ok {
			t.Errorf("SlugOf(%q) = %q, true; want false", name, got)
		}
	}
}
