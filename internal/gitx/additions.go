package gitx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Addition is a file that a commit added.
type Addition struct {
	Path, SHA string
	Time      time.Time
}

// Additions lists, newest first, the files under dir that each commit on the
// first-parent history of tip added against its first parent, one entry per
// commit and file. Commits reachable from since are left out when it is not
// empty. The root commit is compared with nothing.
func (g Git) Additions(tip, since, dir string) ([]Addition, error) {
	args := []string{"-c", "core.quotePath=false", "log", "--root", "--first-parent", "--diff-merges=first-parent",
		"--diff-filter=A", "--name-only", "--format=%x01%H %ct", tip}
	if since != "" {
		args = append(args, "^"+since)
	}
	out, err := g.output(append(args, "--", strings.TrimSuffix(dir, "/")+"/")...)
	if err != nil {
		return nil, err
	}
	var adds []Addition
	var sha string
	var when time.Time
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case line == "":
		case line[0] == 1:
			f := strings.Fields(line[1:])
			if len(f) != 2 {
				return nil, fmt.Errorf("git log: unexpected line %q", line)
			}
			secs, err := strconv.ParseInt(f[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("git log: commit time of %s: %w", f[0], err)
			}
			sha, when = f[0], time.Unix(secs, 0).UTC()
		default:
			adds = append(adds, Addition{Path: line, SHA: sha, Time: when})
		}
	}
	return adds, nil
}

// Tags lists the tags whose object peels to a commit, oldest first by creation
// date, then by name. Name is the full refname and SHA the commit. git fails
// the whole listing when a tag's object is missing, and so does Tags.
func (g Git) Tags() ([]Ref, error) {
	out, err := g.output("for-each-ref", "--sort=refname", "--sort=creatordate",
		"--format=%(refname)%09%(objecttype)%09%(objectname)%09%(*objecttype)%09%(*objectname)", "refs/tags")
	if err != nil {
		return nil, err
	}
	var tags []Ref
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			return nil, fmt.Errorf("git for-each-ref: unexpected line %q", line)
		}
		switch {
		case f[1] == "commit":
			tags = append(tags, Ref{Name: f[0], SHA: f[2]})
		case f[1] == "tag" && f[3] == "commit":
			tags = append(tags, Ref{Name: f[0], SHA: f[4]})
		}
	}
	return tags, nil
}
