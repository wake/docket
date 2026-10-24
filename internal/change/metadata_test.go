package change

import (
	"strings"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	m, err := ParseMetadata([]byte("schema: docket\ncreated: 2026-01-02\nx-kind: fix\nx-source: \"a: b\"\nx-base: abc123\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Metadata{Schema: "docket", Created: "2026-01-02", Kind: "fix", Source: "a: b", Base: "abc123"}
	if m != want {
		t.Fatalf("got %+v, want %+v", m, want)
	}
	m, err = ParseMetadata([]byte("schema: docket\n"))
	if err != nil || m.Base != "" {
		t.Fatalf("missing key: %+v, %v", m, err)
	}
	for _, bad := range []string{"schema: [\n", "- a\n- b\n", "", "null\n", "abc\n", "- a\n"} {
		if _, err := ParseMetadata([]byte(bad)); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}

func TestSetBase(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantErr        bool
	}{
		{"plain", "schema: docket\nx-base: old\nx-kind: fix\n", "schema: docket\nx-base: new\nx-kind: fix\n", false},
		{"crlf", "schema: docket\r\nx-base: old\r\nx-kind: fix\r\n", "schema: docket\r\nx-base: new\r\nx-kind: fix\r\n", false},
		{"quoted", "x-base: \"abc\"\n", "x-base: new\n", false},
		{"no final newline", "schema: docket\nx-base: old", "schema: docket\nx-base: new", false},
		{"missing", "schema: docket\n", "", true},
		{"twice", "x-base: a\nx-base: b\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetBase([]byte(tt.in), "new")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if err == nil && string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetBaseKeepsInput(t *testing.T) {
	in := []byte("x-base: old\n")
	if _, err := SetBase(in, "new"); err != nil || !strings.HasPrefix(string(in), "x-base: old") {
		t.Fatalf("input changed or error: %q %v", in, err)
	}
}
