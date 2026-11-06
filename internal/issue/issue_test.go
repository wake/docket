package issue_test

import (
	"reflect"
	"regexp"
	"testing"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/issue"
)

func TestTemplateSections(t *testing.T) {
	b, err := docket.Shipped.ReadFile("shipped/issue/template.md")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range regexp.MustCompile(`(?m)^## (.+?)\s*$`).FindAllStringSubmatch(string(b), -1) {
		got = append(got, m[1])
	}
	if !reflect.DeepEqual(got, issue.Sections) {
		t.Errorf("sections = %v, want %v", got, issue.Sections)
	}
}

func TestPath(t *testing.T) {
	if got := issue.Path("2026-11-06-x"); got != "docket/issues/2026-11-06-x.md" {
		t.Errorf("Path = %q", got)
	}
}
