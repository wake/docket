package record

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

func TestCommitsFor(t *testing.T) {
	cs := []gitx.Commit{{SHA: "1", Record: []string{"a"}}, {SHA: "2", Record: []string{"b"}}, {SHA: "3", Record: []string{"a", "b"}}}
	got := CommitsFor("a", cs)
	if len(got) != 2 || got[0].SHA != "1" || got[1].SHA != "3" {
		t.Fatalf("got %+v", got)
	}
}

func TestSlugOf(t *testing.T) {
	if got, ok := SlugOf("2026-10-30-fix-x.md"); !ok || got != "2026-10-30-fix-x" {
		t.Fatalf("SlugOf = %q, %v", got, ok)
	}
	for _, name := range []string{"README.md", "2026-10-30-fix-x.txt", ".md", "2026-10-30-fix-x", "2026-10-30.md", "2026-10-30-.md", "notes.md", "2026-10-30xy.md"} {
		if got, ok := SlugOf(name); ok {
			t.Errorf("SlugOf(%q) = %q, true; want false", name, got)
		}
	}
}

func TestValidateName(t *testing.T) {
	if err := ValidateName("fix-x"); err != nil {
		t.Errorf("fix-x: %v", err)
	}
	if err := ValidateName("2026-10-30-fix-x"); err == nil || !strings.Contains(err.Error(), "dkt record adds the date") {
		t.Errorf("dated name: %v", err)
	}
	if err := ValidateName("Fix_X"); err == nil {
		t.Error("Fix_X passed")
	}
	if err := ValidateName(strings.Repeat("a", 189)); err != nil {
		t.Errorf("189 characters: %v", err)
	}
	if err := ValidateName(strings.Repeat("a", 190)); err == nil || !strings.Contains(err.Error(), "200") {
		t.Errorf("190 characters: %v", err)
	}
}

func TestSlugUsesTheTimesLocation(t *testing.T) {
	now := time.Date(2026, 10, 31, 1, 0, 0, 0, time.FixedZone("", 8*3600))
	if got := Slug(now, "fix-x"); got != "2026-10-31-fix-x" {
		t.Errorf("Slug = %q", got)
	}
}

func never(string) bool { return false }

func TestCreate(t *testing.T) {
	content := []byte("hello\n")
	read := func(t *testing.T, path string) string {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	t.Run("free", func(t *testing.T) {
		dir := t.TempDir()
		got, err := Create(dir, "x", never, content)
		if err != nil || got != "x" || read(t, filepath.Join(dir, "x.md")) != "hello\n" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("used elsewhere", func(t *testing.T) {
		dir := t.TempDir()
		got, err := Create(dir, "x", func(s string) bool { return s == "x" }, content)
		if err != nil || got != "x-2" || read(t, filepath.Join(dir, "x-2.md")) != "hello\n" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("file exists", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x.md"), []byte("old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Create(dir, "x", never, content)
		if err != nil || got != "x-2" || read(t, filepath.Join(dir, "x.md")) != "old\n" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("used and existing", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x-2.md"), []byte("old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Create(dir, "x", func(s string) bool { return s == "x" }, content)
		if err != nil || got != "x-3" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("symbolic link", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "target.md")
		if err := os.WriteFile(outside, []byte("target\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "x-2.md")); err != nil {
			t.Fatal(err)
		}
		_, err := Create(dir, "x", func(s string) bool { return s == "x" }, content)
		var le *LinkError
		if !errors.As(err, &le) || le.Name != "x-2.md" {
			t.Fatalf("err = %v", err)
		}
		if read(t, outside) != "target\n" {
			t.Error("the link target was written")
		}
		if _, err := os.Lstat(filepath.Join(dir, "x-3.md")); err == nil {
			t.Error("x-3.md was written past the link")
		}
	})
}

func TestCreateAtTheLimit(t *testing.T) {
	dir := t.TempDir()
	slug := strings.Repeat("a", 198)
	got, err := Create(dir, slug, func(s string) bool { return s == slug }, []byte("x"))
	if err != nil || got != slug+"-2" || len(got) != 200 {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, got+".md")); err != nil {
		t.Error(err)
	}
}

func TestCreateTooLong(t *testing.T) {
	dir := t.TempDir()
	slug := strings.Repeat("a", 199)
	_, err := Create(dir, slug, func(s string) bool { return s == slug }, []byte("x"))
	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("err = %v", err)
	}
	if es, _ := os.ReadDir(dir); len(es) != 0 {
		t.Errorf("wrote %d files", len(es))
	}

	missing := filepath.Join(t.TempDir(), "records")
	_, err = Create(missing, slug, func(s string) bool { return s == slug }, []byte("x"))
	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Lstat(missing); err == nil {
		t.Error("the folder was created")
	}
}

func TestCreateMakesTheFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "docket", "records")
	got, err := Create(dir, "x", never, []byte("hello\n"))
	if err != nil || got != "x" {
		t.Fatalf("got %q, %v", got, err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "x.md")); err != nil || string(b) != "hello\n" {
		t.Errorf("%q, %v", b, err)
	}
}

func TestRender(t *testing.T) {
	const tpl = "<!-- c -->\n\n## What\n\n<!-- x -->\n"
	title := "Fix: the #1 timeout"
	b, err := Render(title, "fix", "review of #40", tpl)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	front, rest, ok := strings.Cut(strings.TrimPrefix(s, "---\n"), "---\n\n")
	if !strings.HasPrefix(s, "---\n") || !ok {
		t.Fatalf("no front matter:\n%s", s)
	}
	d, err := change.ParseDocket([]byte(front))
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != title || d.Kind != "fix" || d.Source != "review of #40" || d.Base != "" {
		t.Errorf("docket = %+v", d)
	}
	if rest != tpl {
		t.Errorf("rest = %q, want the template", rest)
	}
}
