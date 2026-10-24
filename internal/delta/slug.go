package delta

import (
	"strings"
	"unicode"
)

// Slug lower-cases name, turns every run of characters that are neither
// letters nor digits into one "-" and trims leading and trailing "-".
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
