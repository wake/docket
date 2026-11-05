package gitx

import (
	"bytes"
	"fmt"
	"strings"
)

// Trailers is one commit and the values of its Change trailer.
type Trailers struct {
	SHA    string
	Change []string
}

// ChangeTrailers lists every commit reachable from HEAD, oldest first, with
// the values of its Change trailer as git reads trailers.
func (g Git) ChangeTrailers() ([]Trailers, error) {
	out, err := g.output("log", "-z", "--reverse", "--format=%H%x00%(trailers:key=Change,valueonly,unfold)", "HEAD")
	if err != nil {
		return nil, err
	}
	fields := bytes.Split(out, []byte{0})
	if n := len(fields); n > 0 && len(fields[n-1]) == 0 {
		fields = fields[:n-1]
	}
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("git log: unexpected output: %d fields", len(fields))
	}
	var res []Trailers
	for i := 0; i < len(fields); i += 2 {
		t := Trailers{SHA: strings.TrimSpace(string(fields[i]))}
		for _, v := range strings.Split(string(fields[i+1]), "\n") {
			if v = strings.TrimSpace(v); v != "" {
				t.Change = append(t.Change, v)
			}
		}
		res = append(res, t)
	}
	return res, nil
}
