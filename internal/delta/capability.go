package delta

import "strings"

// CapabilityPath returns the capability a delta spec belongs to. rel is
// slash-separated and relative to the change folder; ok is true when it is
// specs/<capability-path>/spec.md.
func CapabilityPath(rel string) (string, bool) {
	if !strings.HasPrefix(rel, "specs/") || !strings.HasSuffix(rel, "/spec.md") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(rel, "specs/"), "/spec.md"), true
}
