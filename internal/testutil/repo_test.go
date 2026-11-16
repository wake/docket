package testutil

import (
	"slices"
	"testing"
)

func TestWithoutHome(t *testing.T) {
	got := WithoutHome([]string{"HOME=/h", "PATH=/p", "UserProfile=/u", "USERPROFILE=/u", "home=/x", "HOMEDRIVE=C:"})
	if want := []string{"PATH=/p", "HOMEDRIVE=C:"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
