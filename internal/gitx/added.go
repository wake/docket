package gitx

import (
	"sort"
	"strings"
)

// UncommittedUnder lists the sorted paths below paths with staged, unstaged
// or untracked changes, untracked files one by one; ignored files are left out.
func (g Git) UncommittedUnder(paths ...string) ([]string, error) {
	return g.changedPaths("all", paths...)
}

// StatusRecords maps every path with staged, unstaged or untracked changes to
// its two-letter XY status code, untracked files one by one.
func (g Git) StatusRecords() (map[string]string, error) {
	recs, err := g.status("all")
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(recs))
	for _, rec := range recs {
		if len(rec) > 3 {
			m[rec[3:]] = rec[:2]
		}
	}
	return m, nil
}

// changedPaths returns the sorted paths of every git status record, whatever
// its state.
func (g Git) changedPaths(untracked string, paths ...string) ([]string, error) {
	recs, err := g.status(untracked, paths...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rec := range recs {
		if len(rec) > 3 {
			files = append(files, rec[3:])
		}
	}
	sort.Strings(files)
	return files, nil
}

// status runs git status and returns its records ("XY path"), sorted.
func (g Git) status(untracked string, paths ...string) ([]string, error) {
	args := append([]string{"status", "--porcelain=v1", "-z", "--untracked-files=" + untracked, "--no-renames", "--"}, paths...)
	out, err := g.output(args...)
	if err != nil {
		return nil, err
	}
	return splitNUL(string(out)), nil
}

// splitNUL splits NUL-terminated output into its non-empty records, sorted.
func splitNUL(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, "\x00") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	sort.Strings(parts)
	return parts
}
