// Package workflow locates a user's workflow file, reads it, and reports its
// state against the shipped template. It never writes the file.
package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/wake/docket/internal/markdown"
)

// ErrNoHome means no absolute home directory could be found.
var ErrNoHome = errors.New("cannot find the home directory: HOME is not an absolute path and the user lookup gave none")

// Status is the state of the workflow file.
type Status string

const (
	OK         Status = "ok"
	Missing    Status = "missing"
	Incomplete Status = "incomplete"
	Stale      Status = "stale"
)

// Sections are the headings a workflow file must have, without the "## ".
var Sections = []string{"Full review", "Light review", "Check", "Direct merge"}

// Report is what Inspect finds in a workflow file. Inspect never sets Status
// to Missing: a missing file is the caller's to report.
type Report struct {
	Status          Status
	Recorded        *string // the template hash the file records; nil when none
	MissingSections []string
}

// lookup returns the last value of key in env, or "" and false.
func lookup(env []string, key string) (string, bool) {
	v, found := "", false
	for _, e := range env {
		if k, val, ok := strings.Cut(e, "="); ok && k == key {
			v, found = val, true
		}
	}
	return v, found
}

// Locate returns the path of the workflow file: under XDG_CONFIG_HOME when
// that is an absolute path, else under the home's .config. The home is HOME
// when absolute, else the result of userHome (which may be nil). It reads only
// env; the last value of a repeated variable counts.
func Locate(env []string, userHome func() (string, error)) (string, error) {
	if x, _ := lookup(env, "XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "docket", "workflow.md"), nil
	}
	home, _ := lookup(env, "HOME")
	if !filepath.IsAbs(home) {
		home = ""
		if userHome != nil {
			if h, err := userHome(); err == nil && filepath.IsAbs(h) {
				home = h
			}
		}
	}
	if home == "" {
		return "", ErrNoHome
	}
	return filepath.Join(home, ".config", "docket", "workflow.md"), nil
}

// Hash returns the first 12 lowercase hexadecimal digits of the SHA-256 of text.
func Hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:12]
}

// Read reads the file at path. A path that does not exist gives found false
// and a nil error. A dangling link anywhere in the path (the deepest existing
// entry is a link whose target does not exist), a directory or any other
// failure is an error.
func Read(path string) (b []byte, found bool, err error) {
	b, err = os.ReadFile(path)
	if err == nil {
		return b, true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		p := path
		for {
			if _, lerr := os.Lstat(p); lerr == nil {
				if _, serr := os.Stat(p); serr != nil && errors.Is(serr, fs.ErrNotExist) {
					return nil, false, fmt.Errorf("%s is a link whose target does not exist", p)
				}
				break
			}
			parent := filepath.Dir(p)
			if parent == p {
				break
			}
			p = parent
		}
		return nil, false, nil
	}
	return nil, false, err
}

func trimLine(s string) string { return strings.TrimRight(s, " \t") }

// Inspect reports the state of the workflow file contents b against the
// shipped template hash. A status is decided in the order incomplete, stale, ok.
func Inspect(b []byte, shipped string) Report {
	lines := markdown.Lines(b)
	r := Report{MissingSections: []string{}}
	r.Recorded = recorded(lines)
	have := map[string]bool{}
	for _, l := range lines {
		have[trimLine(l)] = true
	}
	for _, s := range Sections {
		if !have["## "+s] {
			r.MissingSections = append(r.MissingSections, s)
		}
	}
	switch {
	case len(r.MissingSections) > 0:
		r.Status = Incomplete
	case r.Recorded == nil || *r.Recorded != shipped:
		r.Status = Stale
	default:
		r.Status = OK
	}
	return r
}

// recorded reads the template hash from the frontmatter, or nil.
func recorded(lines []string) *string {
	if len(lines) == 0 || trimLine(lines[0]) != "---" {
		return nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if trimLine(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil
	}
	var fm struct {
		Template yaml.Node `yaml:"template"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return nil
	}
	n := fm.Template
	if n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		return nil
	}
	v := n.Value
	return &v
}
