package delta

import "testing"

func TestCapabilityPath(t *testing.T) {
	cases := []struct {
		rel  string
		want string
		ok   bool
	}{
		{"specs/booking/spec.md", "booking", true},
		{"specs/a/b/spec.md", "a/b", true},
		{"specs/spec.md", "spec.md", true},
		{"proposal.md", "", false},
		{"specs/booking/notes.md", "", false},
		{"specs/booking/spec.md.bak", "", false},
		{"other/booking/spec.md", "", false},
	}
	for _, c := range cases {
		got, ok := CapabilityPath(c.rel)
		if got != c.want || ok != c.ok {
			t.Errorf("CapabilityPath(%q) = %q, %v; want %q, %v", c.rel, got, ok, c.want, c.ok)
		}
	}
}
