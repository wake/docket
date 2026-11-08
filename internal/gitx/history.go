package gitx

import (
	"fmt"
	"strings"
)

// Trailer is one key and value of a commit's trailers.
type Trailer struct{ Key, Value string }

// Commit is one commit with what a snapshot reads of it. Time is %cI.
type Commit struct {
	SHA      string
	Parents  []string
	Time     string
	Subject  string
	Trailers []Trailer // the Change and Record trailers of the last paragraph
}

// History lists every commit reachable from revs, each once, in one git log.
// With no revs it returns nil.
func (g Git) History(revs ...string) ([]Commit, error) {
	if len(revs) == 0 {
		return nil, nil
	}
	args := append([]string{"log", "-z", "--format=%H%x00%P%x00%cI%x00%s%x00%(trailers:key=Change,key=Record,unfold)"}, revs...)
	out, err := g.output(append(args, "--")...)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(out), "\x00")
	if n := len(fields); fields[n-1] == "" {
		fields = fields[:n-1]
	}
	if len(fields)%5 != 0 {
		return nil, fmt.Errorf("git log: unexpected output: %d fields", len(fields))
	}
	var res []Commit
	for i := 0; i < len(fields); i += 5 {
		c := Commit{
			SHA:     strings.TrimSpace(fields[i]),
			Parents: strings.Fields(fields[i+1]),
			Time:    fields[i+2],
			Subject: fields[i+3],
		}
		for _, l := range strings.Split(fields[i+4], "\n") {
			if k, v, ok := strings.Cut(l, ":"); ok {
				c.Trailers = append(c.Trailers, Trailer{strings.TrimSpace(k), strings.TrimSpace(v)})
			}
		}
		res = append(res, c)
	}
	return res, nil
}

// Tag is a tag peeled to its commit; Time is the tag's creation time.
type Tag struct{ Name, Commit, Time string }

// Tags lists refs/tags, oldest created first, in one for-each-ref. A tag that
// does not point at a commit (directly or through one tag object) is left out.
func (g Git) Tags() ([]Tag, error) {
	out, err := g.Run("for-each-ref", "--sort=creatordate",
		"--format=%(refname:lstrip=2)%09%(objecttype)%09%(objectname)%09%(*objecttype)%09%(*objectname)%09%(creatordate:iso-strict)", "refs/tags")
	if err != nil || out == "" {
		return nil, err
	}
	var tags []Tag
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 6 {
			return nil, fmt.Errorf("git for-each-ref: unexpected line %q", line)
		}
		switch {
		case f[1] == "commit":
			tags = append(tags, Tag{f[0], f[2], f[5]})
		case f[1] == "tag" && f[3] == "commit":
			tags = append(tags, Tag{f[0], f[4], f[5]})
		}
	}
	return tags, nil
}

// AddedDates maps each path under dir that a commit reachable from rev added
// to the date (YYYY-MM-DD, in the commit's own offset) of the oldest such
// commit. A rename counts as an add.
func (g Git) AddedDates(rev, dir string) (map[string]string, error) {
	out, err := g.output("log", "-z", "--reverse", "--no-renames", "--diff-filter=A", "--name-only", "--format=%x01%cI%x01", rev, "--", dir)
	if err != nil {
		return nil, err
	}
	res := map[string]string{}
	parts := strings.Split(string(out), "\x01")
	for i := 1; i+1 < len(parts); i += 2 {
		for _, p := range strings.Split(parts[i+1], "\x00") {
			if p = strings.TrimPrefix(p, "\n"); p != "" && len(parts[i]) >= 10 {
				if _, ok := res[p]; !ok {
					res[p] = parts[i][:10]
				}
			}
		}
	}
	return res, nil
}
