// Package userdirs finds the directories of the person running dkt. It reads
// only the environment it is given.
package userdirs

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrNoHome means no absolute home directory could be found.
var ErrNoHome = errors.New("cannot find the home directory: HOME is not an absolute path and the user lookup gave none")

// Lookup returns the last value of key in env, or "" and false.
func Lookup(env []string, key string) (string, bool) {
	v, found := "", false
	for _, e := range env {
		if k, val, ok := strings.Cut(e, "="); ok && k == key {
			v, found = val, true
		}
	}
	return v, found
}

// Home returns the home directory: HOME when absolute, else the result of
// userHome (which may be nil) when absolute, else ErrNoHome.
func Home(env []string, userHome func() (string, error)) (string, error) {
	home, _ := Lookup(env, "HOME")
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
	return home, nil
}

// RelativeError says an environment variable that must be an absolute path
// is not one.
type RelativeError struct{ Name, Value string }

func (e *RelativeError) Error() string {
	return e.Name + " is not an absolute path: " + e.Value
}

// DataHome returns the user data directory: XDG_DATA_HOME when it is not
// empty (a *RelativeError when it is relative), else <home>/.local/share.
func DataHome(env []string, userHome func() (string, error)) (string, error) {
	if x, _ := Lookup(env, "XDG_DATA_HOME"); x != "" {
		if !filepath.IsAbs(x) {
			return "", &RelativeError{"XDG_DATA_HOME", x}
		}
		return x, nil
	}
	home, err := Home(env, userHome)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
