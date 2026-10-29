package change

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

func TestReadMetadataFromDocket(t *testing.T) {
	files := map[string][]byte{
		"docket.yaml":    []byte("title: Booking waitlist\nkind: feature\nsource: issue 12\nbase: " + sha + "\n"),
		".openspec.yaml": []byte("schema: docket\ncreated: 2026-01-02\n"),
	}
	m, err := ReadMetadata(files)
	if err != nil {
		t.Fatal(err)
	}
	want := Metadata{Title: "Booking waitlist", Kind: "feature", Source: "issue 12", Base: sha, Created: "2026-01-02", File: "docket.yaml"}
	if m != want {
		t.Fatalf("got %+v, want %+v", m, want)
	}
}

func TestReadMetadataLegacy(t *testing.T) {
	m, err := ReadMetadata(map[string][]byte{
		".openspec.yaml": []byte("schema: docket\ncreated: 2026-01-02\nx-kind: fix\nx-source: s\nx-base: " + sha + "\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Metadata{Kind: "fix", Source: "s", Base: sha, Created: "2026-01-02", File: ".openspec.yaml"}
	if m != want {
		t.Fatalf("got %+v, want %+v", m, want)
	}
}

func TestReadMetadataBothFiles(t *testing.T) {
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	m, err := ReadMetadata(map[string][]byte{
		"docket.yaml":    []byte("title: T\nbase: " + a + "\n"),
		".openspec.yaml": []byte("x-base: " + b + "\n"),
	})
	if err != nil || m.Base != a {
		t.Fatalf("got %+v, %v", m, err)
	}
}

func TestReadMetadataErrors(t *testing.T) {
	tests := []struct {
		name  string
		files map[string][]byte
		file  string
	}{
		{"docket not a mapping", map[string][]byte{"docket.yaml": []byte("- a\n")}, "docket.yaml"},
		{"no files", map[string][]byte{}, "docket.yaml"},
		{"openspec invalid", map[string][]byte{".openspec.yaml": []byte("schema: [\n")}, ".openspec.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ReadMetadata(tt.files)
			if err == nil {
				t.Fatal("want error")
			}
			if m.File != tt.file {
				t.Fatalf("File = %q, want %q", m.File, tt.file)
			}
		})
	}
	if _, err := ReadMetadata(map[string][]byte{}); err == nil || err.Error() != "docket.yaml is missing" {
		t.Fatalf("err = %v", err)
	}
}

func TestReadMetadataBrokenOpenSpecBesideDocket(t *testing.T) {
	m, err := ReadMetadata(map[string][]byte{"docket.yaml": []byte("title: T\n"), ".openspec.yaml": []byte("schema: [\n")})
	if err != nil || m.Title != "T" || m.Created != "" {
		t.Fatalf("got %+v, %v", m, err)
	}
}

func TestParseDocketTitleForm(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"title: |\n  a\n  b\n", "", false},
		{"title: \"a\\rb\"\n", "", false},
		{"title: \"   \"\n", "", false},
		{"title: 預約候補\n", "預約候補", false},
		{"title: [a]\n", "", true},
	}
	for _, tt := range tests {
		d, err := ParseDocket([]byte(tt.in))
		if (err != nil) != tt.wantErr || d.Title != tt.want {
			t.Errorf("%q: got %q, %v", tt.in, d.Title, err)
		}
	}
}

func TestValidTitle(t *testing.T) {
	for s, want := range map[string]bool{"": false, "   ": false, "a\nb": false, "a\rb": false, "Booking waitlist": true, "預約候補": true} {
		if got := ValidTitle(s); got != want {
			t.Errorf("ValidTitle(%q) = %v", s, got)
		}
	}
}

func TestCreateDocketRoundTrip(t *testing.T) {
	for _, title := range []string{"Booking: waitlist #2", "'quoted'", "- dash", "預約候補"} {
		p := filepath.Join(t.TempDir(), "docket.yaml")
		d := Docket{Title: title, Kind: "feature", Source: "issue 1", Base: sha}
		if err := CreateDocket(p, d); err != nil {
			t.Fatalf("%q: %v", title, err)
		}
		b, _ := os.ReadFile(p)
		got, err := ParseDocket(b)
		if err != nil || got != d {
			t.Fatalf("%q: got %+v, %v", title, got, err)
		}
		s := string(b)
		if !strings.HasPrefix(s, "title:") {
			t.Fatalf("file = %q", s)
		}
		j, k, l := strings.Index(s, "\nkind:"), strings.Index(s, "\nsource:"), strings.Index(s, "\nbase:")
		if !(0 < j && j < k && k < l) {
			t.Fatalf("key order: %q", s)
		}
		if err := CreateDocket(p, d); !errors.Is(err, fs.ErrExist) {
			t.Fatalf("second create: %v", err)
		}
	}
}

func TestCreateDocketSymlinkNotFollowed(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Symlink(target, filepath.Join(dir, "docket.yaml")); err != nil {
		t.Skip(err)
	}
	if err := CreateDocket(filepath.Join(dir, "docket.yaml"), Docket{Title: "T"}); err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("symlink was followed")
	}
}

func TestCreateDocketInvalidTitle(t *testing.T) {
	if err := CreateDocket(filepath.Join(t.TempDir(), "docket.yaml"), Docket{Title: "a\nb"}); err == nil {
		t.Fatal("want error")
	}
}

func TestSetKeyAppend(t *testing.T) {
	tests := []struct{ in, value, want string }{
		{"a: 1", "T", "a: 1\ntitle: T\n"},
		{"a: 1\r\n", "T", "a: 1\r\ntitle: T\r\n"},
		{"title: Old\nb: 2\n", "Booking: x", "title: 'Booking: x'\nb: 2\n"},
	}
	for _, tt := range tests {
		got, err := SetKey([]byte(tt.in), "title", tt.value)
		if err != nil || string(got) != tt.want {
			t.Errorf("%q: got %q, %v, want %q", tt.in, got, err, tt.want)
		}
	}
	if _, err := SetKey([]byte("title: a\ntitle: b\n"), "title", "T"); err == nil {
		t.Error("two lines: want error")
	}
	if _, err := SetKey([]byte("a: 1\n"), "title", "a\nb"); err == nil {
		t.Error("multi-line value: want error")
	}
}
