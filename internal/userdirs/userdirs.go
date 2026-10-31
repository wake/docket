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
