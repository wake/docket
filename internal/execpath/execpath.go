// Package execpath holds the rules dkt uses to find a program on PATH, which
// differ between Windows and other systems.
package execpath

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
)

// Getenv returns the value of the last entry for key in env, as a child
// process started with env would see it; on Windows keys compare without case.
func Getenv(env []string, key string) (string, bool) {
	val, found := "", false
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && (k == key || runtime.GOOS == "windows" && strings.EqualFold(k, key)) {
			val, found = v, true
		}
	}
	return val, found
}

// Names returns the file names that stand for the program name, in the order
// to try them: on Windows, the name with each extension in PATHEXT from env
// (".com;.exe;.bat;.cmd" when unset), or the name alone when it already has
// one of them; elsewhere the name alone.
func Names(name string, env []string) []string {
	if runtime.GOOS != "windows" {
		return []string{name}
	}
	exts, ok := Getenv(env, "PATHEXT")
	if !ok || exts == "" {
		exts = ".com;.exe;.bat;.cmd"
	}
	var names []string
	for _, e := range strings.Split(strings.ToLower(exts), ";") {
		if e == "" {
			continue
		}
		if strings.EqualFold(filepath.Ext(name), e) {
			return []string{name}
		}
		names = append(names, name+e)
	}
	return names
}

// MayRun reports whether fi's permissions let it run: an execute bit, except
// on Windows, which has none. The caller checks the kind of file.
func MayRun(fi fs.FileInfo) bool {
	return runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0
}
