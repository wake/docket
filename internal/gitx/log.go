package gitx

import (
	"bytes"
	"fmt"
	"strings"
)

// Trailers is one commit and the values of one trailer key.
type Trailers struct {
	SHA    string
	Values []string
}

// ChangeTrailers lists every commit reachable from rev, oldest first, with
// the values of its Change trailer as git reads trailers.
func (g Git) ChangeTrailers(rev string) ([]Trailers, error) { return g.trailers("Change", rev) }

// RecordTrailers is ChangeTrailers for the Record trailer.
func (g Git) RecordTrailers(rev string) ([]Trailers, error) { return g.trailers("Record", rev) }

func (g Git) trailers(key, rev string) ([]Trailers, error) {
	out, err := g.output("log", "-z", "--reverse", "--format=%H%x00%(trailers:key="+key+",valueonly,unfold)", rev)
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
				t.Values = append(t.Values, v)
			}
		}
		res = append(res, t)
	}
	return res, nil
}
