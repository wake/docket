package docket

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	releasesURL = "https://github.com/wake/docket/releases"
	fakeDkt     = "#!/bin/sh\necho 'dkt v9.9.9'\n"
)

// installEnv is a home directory, a directory of fake commands and a directory
// of fake releases, for running install.sh.
type installEnv struct{ home, fake, releases string }

func newInstallEnv(t *testing.T) installEnv {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell script")
	}
	root := t.TempDir()
	e := installEnv{filepath.Join(root, "home dir"), filepath.Join(root, "fake"), filepath.Join(root, "releases")}
	for _, d := range []string{e.home, e.releases} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFiles(t, e.fake, map[string]string{
		"curl": `#!/bin/sh
while [ $# -gt 0 ]; do
	case "$1" in
	-o) out=$2; shift ;;
	-*) ;;
	*) url=$1 ;;
	esac
	shift
done
echo "$url" >> "$FAKE_RELEASES/requests"
if [ -n "${FAKE_CURL_SLEEP:-}" ]; then sleep "$FAKE_CURL_SLEEP"; fi
path=${url#https://github.com/wake/docket/releases/}
if [ -f "$FAKE_RELEASES/$path" ]; then
	cp "$FAKE_RELEASES/$path" "$out"
else
	echo "curl: (22) The requested URL returned error: 404 $url" >&2
	exit 22
fi
`,
		"uname": `#!/bin/sh
case "$1" in
-s) echo "$FAKE_OS" ;;
-m) echo "$FAKE_ARCH" ;;
*) exit 1 ;;
esac
`,
	})
	return e
}

func (e installEnv) cmd(input []byte, env ...string) *exec.Cmd {
	c := exec.Command("sh")
	c.Stdin = bytes.NewReader(input)
	c.Env = append([]string{
		"HOME=" + e.home,
		"PATH=" + e.fake + ":/usr/bin:/bin",
		"FAKE_RELEASES=" + e.releases,
		"FAKE_OS=Linux",
		"FAKE_ARCH=x86_64",
	}, env...)
	return c
}

func (e installEnv) runInput(input []byte, env ...string) (stdout, stderr string, err error) {
	c := e.cmd(input, env...)
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	err = c.Run()
	return out.String(), errb.String(), err
}

func (e installEnv) run(t *testing.T, env ...string) (stdout, stderr string, err error) {
	t.Helper()
	script, rerr := os.ReadFile("install.sh")
	if rerr != nil {
		t.Fatal(rerr)
	}
	return e.runInput(script, env...)
}

// publish writes an executable and a matching checksums.txt under dir.
func (e installEnv) publish(t *testing.T, dir, name, body string) {
	t.Helper()
	sum := fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(body)), name)
	writeFiles(t, filepath.Join(e.releases, dir), map[string]string{name: body, "checksums.txt": sum})
}

func (e installEnv) bin() string { return filepath.Join(e.home, ".local", "bin") }

// earlier installs a dkt that a failed run must leave alone.
func (e installEnv) earlier(t *testing.T) {
	t.Helper()
	writeFiles(t, e.bin(), map[string]string{"dkt": "earlier\n"})
}

func (e installEnv) requests(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(e.releases, "requests"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(b))
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, en := range entries {
		names = append(names, en.Name())
	}
	return names
}

func (e installEnv) wantOnlyDkt(t *testing.T) {
	t.Helper()
	if got := listDir(t, e.bin()); len(got) != 1 || got[0] != "dkt" {
		t.Errorf("~/.local/bin holds %v, want only dkt", got)
	}
}

func (e installEnv) wantFile(t *testing.T, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(e.bin(), name))
	if err != nil || string(got) != want {
		t.Errorf("%s = %q, %v; want %q", name, got, err, want)
	}
}

func TestInstallLatest(t *testing.T) {
	rows := []struct{ os, arch, name string }{
		{"Darwin", "arm64", "dkt-darwin-arm64"},
		{"Darwin", "x86_64", "dkt-darwin-amd64"},
		{"Linux", "aarch64", "dkt-linux-arm64"},
		{"Linux", "arm64", "dkt-linux-arm64"},
		{"Linux", "x86_64", "dkt-linux-amd64"},
		{"Linux", "amd64", "dkt-linux-amd64"},
	}
	for _, r := range rows {
		t.Run(r.os+"/"+r.arch, func(t *testing.T) {
			e := newInstallEnv(t)
			e.publish(t, "latest/download", r.name, fakeDkt)
			stdout, stderr, err := e.run(t, "FAKE_OS="+r.os, "FAKE_ARCH="+r.arch)
			if err != nil {
				t.Fatalf("%v\nstderr: %s", err, stderr)
			}
			base := releasesURL + "/latest/download/"
			want := []string{base + r.name, base + "checksums.txt"}
			if got := e.requests(t); strings.Join(got, " ") != strings.Join(want, " ") {
				t.Errorf("requests = %v, want %v", got, want)
			}
			e.wantFile(t, "dkt", fakeDkt)
			e.wantOnlyDkt(t)
			target := filepath.Join(e.bin(), "dkt")
			if fi, err := os.Stat(target); err != nil || fi.Mode()&0o100 == 0 {
				t.Errorf("mode = %v, %v; want the owner's execute bit", fi, err)
			}
			if out, err := exec.Command(target, "--version").Output(); err != nil || string(out) != "dkt v9.9.9\n" {
				t.Errorf("installed dkt prints %q, %v", out, err)
			}
			if !strings.Contains(stdout, target) || !strings.Contains(stdout, "dkt v9.9.9") {
				t.Errorf("stdout = %q, want the path and the version", stdout)
			}
		})
	}
}

func TestInstallReplacesEarlier(t *testing.T) {
	e := newInstallEnv(t)
	e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
	e.earlier(t)
	if _, stderr, err := e.run(t); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr)
	}
	e.wantFile(t, "dkt", fakeDkt)
	e.wantOnlyDkt(t)
}

func TestInstallReplacesLink(t *testing.T) {
	e := newInstallEnv(t)
	e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(e.bin(), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(e.bin(), "dkt")
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := e.run(t); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr)
	}
	if fi, err := os.Lstat(target); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("target = %v, %v; want a regular file", fi, err)
	}
	e.wantFile(t, "dkt", fakeDkt)
	if b, _ := os.ReadFile(outside); string(b) != "outside\n" {
		t.Errorf("the link's target changed to %q", b)
	}
}

func TestInstallVersion(t *testing.T) {
	for _, c := range []struct{ version, dir string }{
		{"v0.16.0", "download/v0.16.0"},
		{"0.16.0", "download/v0.16.0"},
		{"", "latest/download"},
	} {
		t.Run("["+c.version+"]", func(t *testing.T) {
			e := newInstallEnv(t)
			e.publish(t, c.dir, "dkt-linux-amd64", fakeDkt)
			if _, stderr, err := e.run(t, "DKT_VERSION="+c.version); err != nil {
				t.Fatalf("%v\nstderr: %s", err, stderr)
			}
			got := e.requests(t)
			if len(got) != 2 {
				t.Fatalf("requests = %v, want 2", got)
			}
			for _, u := range got {
				if !strings.HasPrefix(u, releasesURL+"/"+c.dir+"/") {
					t.Errorf("request %s is not under %s/", u, c.dir)
				}
			}
		})
	}
}

func TestInstallRefuses(t *testing.T) {
	const name = "dkt-linux-amd64"
	latest := releasesURL + "/latest/download/"
	cases := []struct {
		name       string
		env        []string
		setup      func(t *testing.T, e installEnv)
		want       []string
		noRequests bool
	}{
		{"wrong digest", nil, func(t *testing.T, e installEnv) {
			e.publish(t, "latest/download", name, fakeDkt)
			writeFiles(t, filepath.Join(e.releases, "latest/download"), map[string]string{
				"checksums.txt": strings.Repeat("0", 64) + "  " + name + "\n"})
		}, []string{"SHA-256"}, false},
		{"no entry", nil, func(t *testing.T, e installEnv) {
			e.publish(t, "latest/download", name, fakeDkt)
			writeFiles(t, filepath.Join(e.releases, "latest/download"), map[string]string{
				"checksums.txt": strings.Repeat("0", 64) + "  other\n"})
		}, []string{"checksums.txt"}, false},
		{"executable not published", nil, func(t *testing.T, e installEnv) {
			e.publish(t, "latest/download", name, fakeDkt)
			os.Remove(filepath.Join(e.releases, "latest/download", name))
		}, []string{latest + name}, false},
		{"checksums not published", nil, func(t *testing.T, e installEnv) {
			e.publish(t, "latest/download", name, fakeDkt)
			os.Remove(filepath.Join(e.releases, "latest/download", "checksums.txt"))
		}, []string{latest + "checksums.txt"}, false},
		{"no such release", []string{"DKT_VERSION=v0.1.0"}, func(t *testing.T, e installEnv) {}, []string{"download/v0.1.0/"}, false},
		{"FreeBSD", []string{"FAKE_OS=FreeBSD"}, func(t *testing.T, e installEnv) {}, []string{"FreeBSD", releasesURL}, true},
		{"Windows shell", []string{"FAKE_OS=MINGW64_NT-10.0"}, func(t *testing.T, e installEnv) {}, []string{"MINGW64_NT-10.0", releasesURL}, true},
		{"riscv64", []string{"FAKE_ARCH=riscv64"}, func(t *testing.T, e installEnv) {}, []string{"riscv64", releasesURL}, true},
		{"does not run", nil, func(t *testing.T, e installEnv) {
			e.publish(t, "latest/download", name, "#!/bin/sh\nexit 3\n")
		}, []string{"does not run"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newInstallEnv(t)
			c.setup(t, e)
			e.earlier(t)
			_, stderr, err := e.run(t, c.env...)
			if err == nil {
				t.Error("exit 0, want non-zero")
			}
			for _, w := range c.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr %q lacks %q", stderr, w)
				}
			}
			if c.noRequests && e.requests(t) != nil {
				t.Errorf("requests = %v, want none", e.requests(t))
			}
			e.wantFile(t, "dkt", "earlier\n")
			e.wantOnlyDkt(t)
		})
	}
}

func TestInstallTargetIsDirectory(t *testing.T) {
	e := newInstallEnv(t)
	e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
	writeFiles(t, filepath.Join(e.bin(), "dkt"), map[string]string{"keep": "x"})
	_, stderr, err := e.run(t)
	if err == nil || !strings.Contains(stderr, "is a directory") {
		t.Errorf("err = %v, stderr = %q; want failure saying is a directory", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(e.bin(), "dkt", "keep")); err != nil {
		t.Error(err)
	}
}

func TestInstallDigestTools(t *testing.T) {
	for _, c := range []struct {
		name  string
		tools []string
		ok    bool
	}{
		{"neither", nil, false},
		{"sha256sum", []string{"sha256sum"}, true},
		{"shasum", []string{"shasum"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newInstallEnv(t)
			e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
			tools := filepath.Join(t.TempDir(), "tools")
			if err := os.MkdirAll(tools, 0o755); err != nil {
				t.Fatal(err)
			}
			all := append([]string{"mktemp", "mkdir", "rm", "chmod", "mv", "awk", "cp", "sleep"}, c.tools...)
			for _, tool := range all {
				p, err := exec.LookPath(tool)
				if err != nil {
					t.Skipf("%s is not on this machine", tool)
				}
				if err := os.Symlink(p, filepath.Join(tools, tool)); err != nil {
					t.Fatal(err)
				}
			}
			_, stderr, err := e.run(t, "PATH="+e.fake+":"+tools)
			if c.ok {
				if err != nil {
					t.Fatalf("%v\nstderr: %s", err, stderr)
				}
				e.wantFile(t, "dkt", fakeDkt)
				return
			}
			if err == nil || !strings.Contains(stderr, "sha256sum") {
				t.Errorf("err = %v, stderr = %q; want failure naming sha256sum", err, stderr)
			}
			if e.requests(t) != nil {
				t.Errorf("requests = %v, want none", e.requests(t))
			}
		})
	}
}

func TestInstallInterrupted(t *testing.T) {
	for _, s := range []syscall.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM} {
		t.Run(s.String(), func(t *testing.T) {
			if signal.Ignored(s) {
				t.Skipf("the test process ignores %v, and sh would inherit that", s)
			}
			e := newInstallEnv(t)
			e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
			e.earlier(t)
			script, err := os.ReadFile("install.sh")
			if err != nil {
				t.Fatal(err)
			}
			c := e.cmd(script, "FAKE_CURL_SLEEP=1")
			if err := c.Start(); err != nil {
				t.Fatal(err)
			}
			for i := 0; e.requests(t) == nil; i++ {
				if i > 500 {
					c.Process.Kill()
					t.Fatal("the script never reached curl")
				}
				time.Sleep(10 * time.Millisecond)
			}
			c.Process.Signal(s)
			done := make(chan error, 1)
			go func() { done <- c.Wait() }()
			select {
			case err := <-done:
				if err == nil {
					t.Error("exit 0, want non-zero")
				}
			case <-time.After(10 * time.Second):
				c.Process.Kill()
				t.Fatal("the script did not exit")
			}
			e.wantFile(t, "dkt", "earlier\n")
			e.wantOnlyDkt(t)
		})
	}
}

func TestInstallCutShort(t *testing.T) {
	script, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(script, []byte("\nmain\n")) {
		t.Fatal("the last line of install.sh must be main")
	}
	// One environment per worker, because a run per byte is slow on its own.
	const workers = 8
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		e := newInstallEnv(t)
		e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
		wg.Add(1)
		go func() {
			defer wg.Done()
			// len-1 is the end of the last command, where the script is whole.
			for n := 1 + w; n < len(script)-1; n += workers {
				e.runInput(script[:n])
				if got := e.requests(t); got != nil {
					t.Errorf("cut at byte %d downloaded %v", n, got)
					return
				}
				if got := listDir(t, e.home); len(got) != 0 {
					t.Errorf("cut at byte %d left %v in the home directory", n, got)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestInstallPathHint(t *testing.T) {
	e := newInstallEnv(t)
	e.publish(t, "latest/download", "dkt-linux-amd64", fakeDkt)
	stdout, stderr, err := e.run(t)
	if err != nil || !strings.Contains(stdout, "is not on your PATH") {
		t.Errorf("without the directory: err = %v, stdout = %q, stderr = %q", err, stdout, stderr)
	}
	stdout, stderr, err = e.run(t, "PATH="+e.bin()+":"+e.fake+":/usr/bin:/bin")
	if err != nil || strings.Contains(stdout, "PATH") {
		t.Errorf("with the directory: err = %v, stdout = %q, stderr = %q", err, stdout, stderr)
	}
}
