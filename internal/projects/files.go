// Package projects reads and writes the machine-wide index of projects and its settings.
package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/wake/docket/internal/execpath"
	"github.com/wake/docket/internal/missing"
	"github.com/wake/docket/internal/safefile"
)

const isWindows = runtime.GOOS == "windows"

// homeVar names the variable holding the home folder, by os.UserHomeDir's rule; dkt reads
// it from the environment it is given rather than the process's own.
func homeVar(windows bool) string {
	if windows {
		return "USERPROFILE"
	}
	return "HOME"
}

// ErrNoHome means neither the XDG variable nor the home folder's variable gives a usable folder.
var ErrNoHome = errors.New("cannot locate the user folders: set " + homeVar(isWindows))

func xdgFile(env []string, xdg, fallback, name string) (string, error) {
	if v, _ := execpath.Getenv(env, xdg); filepath.IsAbs(v) {
		return filepath.Join(v, "docket", name), nil
	}
	if h, _ := execpath.Getenv(env, homeVar(isWindows)); h != "" {
		return filepath.Join(h, fallback, "docket", name), nil
	}
	return "", ErrNoHome
}

// IndexPath is the index file under the state home.
func IndexPath(env []string) (string, error) {
	return xdgFile(env, "XDG_STATE_HOME", ".local/state", "projects.json")
}

// SessionsDir is the folder of per-session activity records, beside the index.
func SessionsDir(env []string) (string, error) {
	return xdgFile(env, "XDG_STATE_HOME", ".local/state", "sessions")
}

// SettingsPath is the settings file under the config home.
func SettingsPath(env []string) (string, error) {
	return xdgFile(env, "XDG_CONFIG_HOME", ".config", "config.json")
}

// UserWorkflowPath is the user's workflow template, beside the settings file.
func UserWorkflowPath(env []string) (string, error) {
	return xdgFile(env, "XDG_CONFIG_HOME", ".config", "workflow.md")
}

// Entry is one project in the index.
type Entry struct {
	Path    string  `json:"path"`
	Origin  *string `json:"origin"`
	Added   string  `json:"added"`
	Ignored bool    `json:"ignored"`
	// Mode is "sync" or "independent"; empty means independent. Synced maps a workflow
	// section's heading to the hash of the text both files had when it was recorded.
	Mode   string            `json:"mode,omitempty"`
	Synced map[string]string `json:"synced,omitempty"`
}

// Index is the content of the index file.
type Index struct {
	Projects []Entry `json:"projects"`
}

// Stamp formats t as UTC to the second; every added value comes from it.
func Stamp(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }

// Load reads the index; a missing file is an empty index.
func Load(path string) (*Index, error) {
	data, err := safefile.ReadNoFollow(path)
	if missing.Path(err, path) {
		return &Index{Projects: []Entry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var x Index
	if err := json.Unmarshal(data, &x); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if x.Projects == nil {
		return nil, fmt.Errorf("read %s: projects is missing", path)
	}
	var raw struct {
		Projects []map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	seen := map[string]bool{}
	for i, e := range x.Projects {
		for _, k := range []string{"path", "origin", "added", "ignored"} {
			if _, ok := raw.Projects[i][k]; !ok {
				return nil, fmt.Errorf("read %s: an entry has no %s", path, k)
			}
		}
		if seen[e.Path] {
			return nil, fmt.Errorf("read %s: path %q is listed twice", path, e.Path)
		}
		seen[e.Path] = true
		if !filepath.IsAbs(e.Path) {
			return nil, fmt.Errorf("read %s: path %q is not absolute", path, e.Path)
		}
		if _, err := time.Parse(time.RFC3339, e.Added); err != nil || !strings.HasSuffix(e.Added, "Z") {
			return nil, fmt.Errorf("read %s: added %q is not an RFC 3339 UTC time", path, e.Added)
		}
		if e.Mode != "" && e.Mode != "sync" && e.Mode != "independent" {
			return nil, fmt.Errorf("read %s: mode %q is not sync or independent", path, e.Mode)
		}
		if e.Origin != nil {
			o := withoutUserInfo(*e.Origin)
			x.Projects[i].Origin = &o
		}
	}
	return &x, nil
}

// Find returns the position of the entry for path, or -1.
func (x *Index) Find(path string) int {
	for i, e := range x.Projects {
		if e.Path == path {
			return i
		}
	}
	return -1
}

// Save sorts the entries by path and replaces the file.
func (x *Index) Save(path string) error {
	sort.Slice(x.Projects, func(i, j int) bool { return x.Projects[i].Path < x.Projects[j].Path })
	data, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return safefile.Install(path, append(data, '\n'), 0o600, false, path)
}

// DefaultMode reads workflow.default from the settings: sync, independent, or empty when
// the file or the key is missing.
func DefaultMode(settings string) (string, error) {
	data, err := os.ReadFile(settings)
	if missing.Path(err, settings) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", settings, err)
	}
	var s struct {
		Workflow struct {
			Default string `json:"default"`
		} `json:"workflow"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return "", fmt.Errorf("read %s: %w", settings, err)
	}
	if d := s.Workflow.Default; d != "" && d != "sync" && d != "independent" {
		return "", fmt.Errorf("read %s: workflow.default %q is not sync or independent", settings, d)
	}
	return s.Workflow.Default, nil
}

// AutoRegister reads projects.autoRegister from the settings; true when missing.
func AutoRegister(settings string) (bool, error) {
	data, err := os.ReadFile(settings)
	if missing.Path(err, settings) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", settings, err)
	}
	var s struct {
		Projects struct {
			AutoRegister *bool `json:"autoRegister"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return false, fmt.Errorf("read %s: %w", settings, err)
	}
	return s.Projects.AutoRegister == nil || *s.Projects.AutoRegister, nil
}
