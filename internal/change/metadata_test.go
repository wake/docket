package change

import (
	"strings"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	m, err := parseMetadata([]byte("schema: docket\ncreated: 2026-01-02\nx-kind: fix\nx-source: \"a: b\"\nx-base: abc123\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Metadata{Created: "2026-01-02", Kind: "fix", Source: "a: b", Base: "abc123", File: OpenSpecFile}
	if m != want {
		t.Fatalf("got %+v, want %+v", m, want)
	}
	m, err = parseMetadata([]byte("schema: docket\n"))
	if err != nil || m.Base != "" {
		t.Fatalf("missing key: %+v, %v", m, err)
	}
	for _, bad := range []string{"schema: [\n", "- a\n- b\n", "", "null\n", "abc\n", "- a\n"} {
		if _, err := parseMetadata([]byte(bad)); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}

func TestSetKey(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantErr        bool
	}{
		{"plain", "schema: docket\nx-base: old\nx-kind: fix\n", "schema: docket\nx-base: new\nx-kind: fix\n", false},
		{"crlf", "schema: docket\r\nx-base: old\r\nx-kind: fix\r\n", "schema: docket\r\nx-base: new\r\nx-kind: fix\r\n", false},
		{"quoted", "x-base: \"abc\"\n", "x-base: new\n", false},
		{"no final newline", "schema: docket\nx-base: old", "schema: docket\nx-base: new", false},
		{"missing appends", "schema: docket\n", "schema: docket\nx-base: new\n", false},
		{"twice", "x-base: a\nx-base: b\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetKey([]byte(tt.in), "x-base", "new")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if err == nil && string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetKeyKeepsInput(t *testing.T) {
	in := []byte("x-base: old\n")
	if _, err := SetKey(in, "x-base", "new"); err != nil || !strings.HasPrefix(string(in), "x-base: old") {
		t.Fatalf("input changed or error: %q %v", in, err)
	}
}

func TestReadSchema(t *testing.T) {
	ok := []struct{ name, in, want string }{
		{"declared", "schema: spec-driven\ncreated: 2026-01-02\n", "spec-driven"},
		{"no key", "created: 2026-01-02\n", DocketSchema},
	}
	for _, tt := range ok {
		got, err := ReadSchema(map[string][]byte{OpenSpecFile: []byte(tt.in)})
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v", tt.name, got, err)
		}
	}
	if got, err := ReadSchema(map[string][]byte{}); err != nil || got != DocketSchema {
		t.Errorf("no file: got %q, %v", got, err)
	}
	for _, in := range []string{"schema:\n", "schema: null\n", "schema: \"\"\n", "schema: 1\n", "schema: true\n", "schema: [a]\n", "- a\n"} {
		if got, err := ReadSchema(map[string][]byte{OpenSpecFile: []byte(in)}); err == nil {
			t.Errorf("%q: want error, got %q", in, got)
		}
	}
}

func TestSourcePath(t *testing.T) {
	for in, want := range map[string]string{
		"docket/ideas/x.md":                            "docket/ideas/x.md",
		"  docket/ideas/x.md  ":                        "docket/ideas/x.md",
		"docket/ideas/x.md, item 1 of docket/order.md": "docket/ideas/x.md",
		"docket/ideas/x.md (order 5)":                  "docket/ideas/x.md",
		"docket/ideas/x.md,":                           "docket/ideas/x.md",
		"see docket/ideas/x.md":                        "see",
		"":                                             "",
		"   ":                                          "",
		"a,,":                                          "a,",
	} {
		if got := SourcePath(in); got != want {
			t.Errorf("SourcePath(%q) = %q, want %q", in, got, want)
		}
	}
}
