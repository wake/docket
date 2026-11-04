package cli

import "testing"

func TestResolveVersion(t *testing.T) {
	cases := []struct{ stamped, bi, want string }{
		{"v0.16.0", "", "v0.16.0"},
		{"v0.16.0", "v0.1.0", "v0.16.0"},
		{"dev", "v0.16.0", "v0.16.0"},
		{"dev", "v0.0.0-20261104074825-4c775bd664cc", "dev"},
		{"dev", "v0.16.0+dirty", "dev"},
		{"dev", "(devel)", "dev"},
		{"dev", "", "dev"},
		{"", "v0.16.0", "v0.16.0"},
		{"", "", "dev"},
	}
	for _, c := range cases {
		if got := ResolveVersion(c.stamped, c.bi); got != c.want {
			t.Errorf("ResolveVersion(%q,%q) = %q, want %q", c.stamped, c.bi, got, c.want)
		}
	}
}
