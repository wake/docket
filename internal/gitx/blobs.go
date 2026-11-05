package gitx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// ReadBlobs reads the files at paths in commit with one git call. A path that
// is missing, is not a file, or holds a newline is left out of the result.
func (g Git) ReadBlobs(commit string, paths []string) (map[string][]byte, error) {
	got := map[string][]byte{}
	var asked []string
	var in strings.Builder
	for _, p := range paths {
		if strings.Contains(p, "\n") {
			continue // cat-file --batch reads one name per line
		}
		asked = append(asked, p)
		in.WriteString(commit + ":" + p + "\n")
	}
	if len(asked) == 0 {
		return got, nil
	}
	out, err := g.outputIn(in.String(), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	rd := bufio.NewReader(bytes.NewReader(out))
	for _, p := range asked {
		line, err := rd.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file: short output at %q: %w", p, err)
		}
		if strings.HasSuffix(strings.TrimSuffix(line, "\n"), " missing") {
			continue // the line echoes the whole name, which may hold spaces
		}
		f := strings.Fields(line)
		if len(f) != 3 {
			return nil, fmt.Errorf("git cat-file: unexpected header %q", line)
		}
		size, err := strconv.Atoi(f[2])
		if err != nil || size < 0 {
			return nil, fmt.Errorf("git cat-file: unexpected header %q", line)
		}
		body := make([]byte, size+1) // the body and its newline
		if _, err := io.ReadFull(rd, body); err != nil {
			return nil, fmt.Errorf("git cat-file: short body at %q: %w", p, err)
		}
		if f[1] == "blob" {
			got[p] = body[:size]
		}
	}
	return got, nil
}

// AddedSince returns the sorted, unique paths under dirs that are new since
// base: committed, staged or untracked, and still in the work tree. Ignored
// files are left out. Paths are relative to the repository root, which is
// where Dir must be.
func (g Git) AddedSince(base string, dirs ...string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, args := range [][]string{
		{"diff", "--name-only", "-z", "--no-renames", "--diff-filter=A", base, "--"},
		{"ls-files", "-z", "--others", "--exclude-standard", "--"},
	} {
		out, err := g.output(append(args, dirs...)...)
		if err != nil {
			return nil, err
		}
		for _, p := range splitNUL(string(out)) {
			seen[p] = true
		}
	}
	res := make([]string, 0, len(seen))
	for p := range seen {
		res = append(res, p)
	}
	sort.Strings(res)
	return res, nil
}
