package conflicts

import (
	"errors"
	"testing"

	"github.com/wake/docket/internal/openspec"
)

func TestParseUnavailableIsFatal(t *testing.T) {
	show := func(dir, slug string) ([]openspec.Delta, error) {
		return nil, &openspec.UnavailableError{Program: "openspec"}
	}
	got, bad, err := Parse([]Copy{{Slug: "a", Sources: []string{"main"}}, {Slug: "b", Sources: []string{"main"}}}, show)
	var ue *openspec.UnavailableError
	if !errors.As(err, &ue) || len(got) != 0 || len(bad) != 0 {
		t.Fatalf("got %+v %+v %v", got, bad, err)
	}
}
