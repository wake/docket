package cli

import "regexp"

var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// ResolveVersion picks the version to report: a stamped one wins, then a
// plain release version from build info (go install at a tag), else "dev".
func ResolveVersion(stamped, buildInfo string) string {
	if stamped != "dev" && stamped != "" {
		return stamped
	}
	if releaseVersion.MatchString(buildInfo) {
		return buildInfo
	}
	return "dev"
}
