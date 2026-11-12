// Package projects reads and writes the machine-wide index of projects and its settings.
package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wake/docket/internal/safefile"
)

// ErrNoHome means neither the XDG variable nor HOME gives a usable folder.
var ErrNoHome = errors.New("cannot locate the user folders: set HOME")

// envValue returns the last value of key in env (the one a child would see).
func envValue(env []string, key string) string {
	val := ""
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k == key {
			val = v
		}
	}
	return val
}

func xdgFile(env []string, xdg, fallback, name string) (string, error) {
	if v := envValue(env, xdg); filepath.IsAbs(v) {
		return filepath.Join(v, "docket", name), nil
	}
	if h := envValue(env, "HOME"); h != "" {
		return filepath.Join(h, fallback, "docket", name), nil
	}
	return "", ErrNoHome
}

// IndexPath is the index file under the state home.
func IndexPath(env []string) (string, error) {
	return xdgFile(env, "XDG_STATE_HOME", ".local/state", "projects.json")
}

// SettingsPath is the settings file under the config home.
func SettingsPath(env []string) (string, error) {
	return xdgFile(env, "XDG_CONFIG_HOME", ".config", "config.json")
}

// Entry is one project in the index.
type Entry struct {
	Path    string  `json:"path"`
	Origin  *string `json:"origin"`
	Added   string  `json:"added"`
	Ignored bool    `json:"ignored"`
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
	if errors.Is(err, fs.ErrNotExist) {
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

// AutoRegister reads projects.autoRegister from the settings; true when missing.
func AutoRegister(settings string) (bool, error) {
	data, err := os.ReadFile(settings)
	if errors.Is(err, fs.ErrNotExist) {
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
