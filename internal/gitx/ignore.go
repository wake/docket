package gitx

import (
	"errors"
	"strconv"
	"strings"
)

// Ignore is one path git ignores and the rule that ignores it.
type Ignore struct {
	Path, Source, Pattern string
	Line                  int
}

// Ignored reports which of paths git ignores, in the order git answers. A
// tracked path is not reported, and neither is one a negated rule un-ignores.
func (g Git) Ignored(paths []string) ([]Ignore, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out, err := g.outputIn(strings.Join(paths, "\x00")+"\x00", "check-ignore", "-v", "-z", "--stdin")
	var ee *ExitError
	if errors.As(err, &ee) && ee.Code == 1 { // none of them is ignored
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var res []Ignore
	for i := 0; i+3 < len(f); i += 4 {
		if strings.HasPrefix(f[i+2], "!") {
			continue
		}
		line, _ := strconv.Atoi(f[i+1])
		res = append(res, Ignore{Source: f[i], Line: line, Pattern: f[i+2], Path: f[i+3]})
	}
	return res, nil
}
