package issue

import (
	"reflect"
	"testing"
)

func TestParseUnsynced(t *testing.T) {
	for v, want := range map[string][]string{"[taken, pr]": {"taken", "pr"}, "[]": {}} {
		got, err := ParseUnsynced(v)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %v %v", v, got, err)
		}
	}
	for _, v := range []string{"taken", "[later]", "[pr, pr]"} {
		if _, err := ParseUnsynced(v); err == nil {
			t.Errorf("%q: no error", v)
		}
	}
}
