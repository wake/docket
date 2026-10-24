package gitx

import (
	"sort"
	"strings"
)

// AddedFiles lists the files under paths that HEAD adds relative to base:
// committed additions only. A rename into the paths counts as an addition.
func (g Git) AddedFiles(base string, paths ...string) ([]string, error) {
	args := append([]string{"diff", "--name-only", "--diff-filter=A", "--no-renames", "-z", base, "HEAD", "--"}, paths...)
	out, err := g.output(args...)
	if err != nil {
		return nil, err
	}
	return splitNUL(string(out)), nil
}

// UncommittedAdded lists the files under paths that are untracked or newly
// added to the index (status A or ? in either column).
func (g Git) UncommittedAdded(paths ...string) ([]string, error) {
	recs, err := g.status("all", paths...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rec := range recs {
		if len(rec) > 3 && strings.ContainsAny(rec[:2], "A?") && rec[1] != 'D' {
			files = append(files, rec[3:])
		}
	}
	sort.Strings(files)
	return files, nil
}

// Uncommitted lists the sorted paths with staged, unstaged or untracked
// changes; ignored files are left out.
func (g Git) Uncommitted() ([]string, error) {
	recs, err := g.status("normal")
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
