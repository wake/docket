// Command fakebin stands in for OpenSpec, git and other programs in tests, on
// every system. testutil builds it once, copies it for each fake and appends a
// JSON payload after Marker; the copy reads the payload from its own file, so
// a copy of the copy behaves the same. See testutil.FakeOpenSpec and
// testutil.FakeProgram.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Marker precedes the payload at the end of the file.
const Marker = "\x00docket-fake-payload\x00"

// Payload says what the fake does; testutil writes it.
type Payload struct {
	Kind string // "openspec" or "program"

	// openspec
	Log     string
	Version string
	Config  string // the openspec/config.yaml that init writes

	// program, in this order: log the arguments; exit with Code when an
	// argument is ExitIfArg; run Real with the arguments and exit with its
	// code; else print Print, remove its own file when RemoveSelf, and exit
	// with Code.
	ExitIfArg  string
	Real       string
	Print      string
	RemoveSelf bool
	Code       int
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		fatal(err)
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		fatal(err)
	}
	i := bytes.LastIndex(b, []byte(Marker))
	if i < 0 {
		fatal(errors.New("no payload"))
	}
	var p Payload
	if err := json.Unmarshal(b[i+len(Marker):], &p); err != nil {
		fatal(err)
	}
	args := os.Args[1:]
	switch p.Kind {
	case "openspec":
		os.Exit(openspec(p, args))
	case "program":
		os.Exit(program(p, exe, args))
	}
	fatal(fmt.Errorf("unknown kind %q", p.Kind))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "fakebin:", err)
	os.Exit(125)
}

// appendTo appends s to the file at p in one write, so concurrent runs do not interleave.
func appendTo(p, s string) {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		fatal(err)
	}
}

func program(p Payload, exe string, args []string) int {
	if p.Log != "" {
		appendTo(p.Log, strings.Join(args, " ")+"\n")
	}
	for _, a := range args {
		if p.ExitIfArg != "" && a == p.ExitIfArg {
			return p.Code
		}
	}
	if p.Real != "" {
		c := exec.Command(p.Real, args...)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := c.Run()
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		if err != nil {
			fatal(err)
		}
		return 0
	}
	fmt.Print(p.Print)
	if p.RemoveSelf {
		// Windows cannot delete a running program's file, but can rename it out of the way.
		gone := exe + ".gone"
		if err := os.Rename(exe, gone); err != nil {
			fatal(err)
		}
		os.Remove(gone)
	}
	return p.Code
}

// openspec does what the earlier shell fake did; testutil.FakeOpenSpec documents it.
func openspec(p Payload, args []string) int {
	line := "OPENSPEC_TELEMETRY=" + os.Getenv("OPENSPEC_TELEMETRY")
	for _, a := range args {
		line += "\t" + a
	}
	appendTo(p.Log, line+"\n")
	appendTo(p.Log+".foo", os.Getenv("FOO")+"\n") // lets a test see that the environment passes through
	for len(args) > 0 && (args[0] == "-y" || strings.HasPrefix(args[0], "@")) {
		args = args[1:] // launcher words such as: -y @scope/pkg@version
	}
	arg := func(n int) string { // $n of the shell fake
		if n <= len(args) {
			return args[n-1]
		}
		return ""
	}
	failWith := func(msg string, code int) int {
		fmt.Fprintln(os.Stderr, msg)
		return code
	}
	must := func(err error) {
		if err != nil {
			fatal(err)
		}
	}
	switch arg(1) {
	case "--version":
		if os.Getenv("FAKE_OPENSPEC_FAIL") == "--version" {
			return failWith("fake failure", 3)
		}
		fmt.Println(p.Version)
	case "init":
		if arg(3) == "nosuch" {
			return failWith("error: unknown tool 'nosuch'", 1)
		}
		for _, d := range []string{"openspec/specs", "openspec/changes/archive", ".claude/skills/openspec-explore"} {
			must(os.MkdirAll(d, 0o755))
		}
		must(os.WriteFile("openspec/config.yaml", []byte(p.Config), 0o644))
		must(os.WriteFile(".claude/skills/openspec-explore/SKILL.md", []byte("Explore mode\n"), 0o644))
		if t := os.Getenv("FAKE_OPENSPEC_TOUCH"); t != "" {
			appendTo(t, "changed\n")
		}
		if c := os.Getenv("FAKE_OPENSPEC_CREATE"); c != "" {
			must(os.MkdirAll(filepath.Dir(c), 0o755))
			must(os.WriteFile(c, []byte("new\n"), 0o644))
		}
	case "show", "validate":
		name := strings.ReplaceAll(strings.Join(args, "_"), "/", "+")
		if dir := os.Getenv("FAKE_OPENSPEC_JSON"); dir != "" {
			if body, err := os.ReadFile(filepath.Join(dir, name+".json")); err == nil {
				os.Stdout.Write(body)
				code := 0
				if c, err := os.ReadFile(filepath.Join(dir, name+".code")); err == nil {
					code, err = strconv.Atoi(strings.TrimSpace(string(c)))
					must(err)
				}
				return code
			}
		}
		fmt.Printf("{\"status\":[{\"severity\":\"error\",\"code\":\"show_error\",\"message\":\"fake: no answer for %s\"}]}\n", name)
		return 1
	case "new":
		slug, schema := arg(3), "docket"
		if arg(4) == "--schema" {
			schema = arg(5)
		}
		if strings.IndexFunc(slug, unicode.IsUpper) >= 0 {
			return failWith("Error: Change name must be lowercase (use kebab-case)", 1)
		}
		dir := filepath.Join("openspec", "changes", slug)
		if _, err := os.Stat(dir); err == nil {
			return failWith("Error: Change '"+slug+"' already exists", 1)
		}
		if s := os.Getenv("FAKE_OPENSPEC_NEW_SLEEP"); s != "" {
			secs, err := strconv.ParseFloat(s, 64)
			must(err)
			time.Sleep(time.Duration(secs * float64(time.Second)))
		}
		if target := os.Getenv("FAKE_OPENSPEC_NEW_DIRLINK"); target != "" {
			must(os.Symlink(target, dir))
			return failWith("fake failure", 2)
		}
		must(os.MkdirAll(dir, 0o755))
		must(os.WriteFile(filepath.Join(dir, ".openspec.yaml"), []byte("schema: "+schema+"\ncreated: 2026-01-01\n"), 0o644))
		if extra := os.Getenv("FAKE_OPENSPEC_NEW_EXTRA"); extra != "" {
			must(os.WriteFile(filepath.Join(dir, extra), []byte("extra\n"), 0o644))
		}
		if target := os.Getenv("FAKE_OPENSPEC_NEW_LINK"); target != "" {
			must(os.Symlink(target, filepath.Join(dir, "proposal.md")))
		}
		if os.Getenv("FAKE_OPENSPEC_NEW_FAIL") == "1" {
			return failWith("fake failure", 2)
		}
	}
	return 0
}
