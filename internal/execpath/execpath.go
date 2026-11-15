// Package execpath holds the rules for finding a program on PATH that differ
// on Windows, for the lookups that read an environment they are given rather
// than the process's own, as exec.LookPath does.
package execpath

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
)

const isWindows = runtime.GOOS == "windows"

// Getenv returns the value of the last entry for key in env, as a process
// started with env sees it: on Windows, names compare without case.
func Getenv(env []string, key string) (string, bool) { return getenv(isWindows, env, key) }

func getenv(windows bool, env []string, key string) (string, bool) {
	val, found := "", false
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && (k == key || windows && strings.EqualFold(k, key)) {
			val, found = v, true
		}
	}
	return val, found
}

// Names returns the file names that stand for the program name, in the order
// to try them: on Windows, the name with each extension in PATHEXT from env
// (".com;.exe;.bat;.cmd" when unset or empty), a dot added where missing as
// exec.LookPath does, or the name alone when it already ends in one of them;
// elsewhere the name alone.
func Names(name string, env []string) []string { return names(isWindows, name, env) }

func names(windows bool, name string, env []string) []string {
	if !windows {
		return []string{name}
	}
	exts, _ := getenv(windows, env, "PATHEXT")
	if exts == "" {
		exts = ".com;.exe;.bat;.cmd"
	}
	var out []string
	for _, e := range strings.Split(strings.ToLower(exts), ";") {
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		if strings.EqualFold(filepath.Ext(name), e) {
			return []string{name}
		}
		out = append(out, name+e)
	}
	return out
}

// MayRun reports whether fi's permissions let it run: an execute bit, except
// on Windows, which has none. The caller checks the kind of file.
func MayRun(fi fs.FileInfo) bool { return mayRun(isWindows, fi.Mode()) }

func mayRun(windows bool, m fs.FileMode) bool { return windows || m.Perm()&0o111 != 0 }
