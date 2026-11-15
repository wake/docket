package activity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Now().UTC().Truncate(time.Second) // file times are real, so "now" is real

func newEnv(t *testing.T) ([]string, string) {
	t.Helper()
	s := t.TempDir()
	return []string{"XDG_STATE_HOME=" + s}, filepath.Join(s, "docket", "sessions")
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var l []string
	for _, e := range es {
		l = append(l, e.Name())
	}
	return l
}

func in(id, cwd string) *strings.Reader {
	return strings.NewReader(`{"session_id":"` + id + `","cwd":"` + cwd + `"}`)
}

func TestWriteLifecycle(t *testing.T) {
	env, dir := newEnv(t)
	w := t.TempDir()
	if err := Write(env, t0, "working", in("a", w), "/x"); err != nil {
		t.Fatal(err)
	}
	fs := files(t, dir)
	if len(fs) != 1 {
		t.Fatalf("%v", fs)
	}
	recs, err := Read(env, t0)
	want, _ := filepath.EvalSymlinks(w)
	if err != nil || len(recs) != 1 || recs[0].Session != "a" || recs[0].Dir != want || recs[0].State != "working" || !recs[0].Time.Equal(t0) {
		t.Fatalf("%+v %v", recs, err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, fs[0]))
	if !strings.Contains(string(raw), `"`+t0.Format(time.RFC3339)+`"`) {
		t.Errorf("time not UTC RFC 3339: %s", raw)
	}
	if err := Write(env, t0.Add(time.Minute), "waiting", in("a", w), "/x"); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); len(got) != 1 || got[0] != fs[0] {
		t.Fatalf("%v", got)
	}
	recs, _ = Read(env, t0.Add(time.Minute))
	if len(recs) != 1 || recs[0].State != "waiting" {
		t.Fatalf("%+v", recs)
	}
	if err := Write(env, t0, "ended", in("a", w), "/x"); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestTwoSessionsOneDirectory(t *testing.T) {
	env, dir := newEnv(t)
	w := t.TempDir()
	for _, id := range []string{"a", "b"} {
		if err := Write(env, t0, "working", in(id, w), "/x"); err != nil {
			t.Fatal(err)
		}
	}
	if len(files(t, dir)) != 2 {
		t.Fatal("want two files")
	}
	Write(env, t0, "ended", in("a", w), "/x")
	recs, _ := Read(env, t0)
	if len(recs) != 1 || recs[0].Session != "b" {
		t.Fatalf("%+v", recs)
	}
}

func TestNoUsableInput(t *testing.T) {
	w := t.TempDir()
	want, _ := filepath.EvalSymlinks(w)
	for name, body := range map[string]string{"nil": "", "text": "not json", "empty object": "{}"} {
		env, dir := newEnv(t)
		for i := 0; i < 2; i++ {
			var err error
			if name == "nil" {
				err = Write(env, t0, "working", nil, w)
			} else {
				err = Write(env, t0, "working", strings.NewReader(body), w)
			}
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
		recs, _ := Read(env, t0)
		if len(files(t, dir)) != 1 || len(recs) != 1 || recs[0].Dir != want || recs[0].Session != "" {
			t.Errorf("%s: %+v", name, recs)
		}
	}
}

func TestDirectoryThroughLink(t *testing.T) {
	env, _ := newEnv(t)
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "l")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	Write(env, t0, "working", in("a", link), "/x")
	recs, _ := Read(env, t0)
	want, _ := filepath.EvalSymlinks(real)
	if len(recs) != 1 || recs[0].Dir != want {
		t.Fatalf("%+v want %s", recs, want)
	}
}

func TestWritePrunes(t *testing.T) {
	env, dir := newEnv(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, age := range map[string]time.Duration{"old.json": 16 * time.Minute, "recent.json": 5 * time.Minute} {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte("x"), 0o600)
		os.Chtimes(p, t0.Add(-age), t0.Add(-age))
	}
	if err := Write(env, t0, "working", in("a", "/w"), "/x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.json")); err == nil {
		t.Error("old file kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "recent.json")); err != nil {
		t.Error("recent file removed")
	}
}

func TestWriteErrors(t *testing.T) {
	env, dir := newEnv(t)
	if Write(env, t0, "busy", nil, "/x") == nil {
		t.Error("unknown state")
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("folder created for unknown state")
	}
	if Write(nil, t0, "working", nil, "/x") == nil {
		t.Error("no home")
	}
	blocked := t.TempDir()
	os.WriteFile(filepath.Join(blocked, "docket"), []byte("f"), 0o600) // a file where the folder must go
	if Write([]string{"XDG_STATE_HOME=" + blocked}, t0, "working", nil, "/x") == nil {
		t.Error("unwritable folder")
	}
}

func TestReadWindowAndSkips(t *testing.T) {
	env, dir := newEnv(t)
	Write(env, t0.Add(-14*time.Minute), "working", in("s0", "/w"), "/x")
	Write(env, t0.Add(-16*time.Minute), "working", in("s1", "/w"), "/x")
	os.WriteFile(filepath.Join(dir, "junk.json"), []byte("nope"), 0o600)
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"session":"x","dir":"/w","state":"busy","time":"`+t0.Format(time.RFC3339)+`"}`), 0o600)
	os.Symlink(filepath.Join(dir, "junk.json"), filepath.Join(dir, "link.json"))
	os.WriteFile(filepath.Join(dir, "locked.json"), []byte("{}"), 0o000)
	before := strings.Join(files(t, dir), ",")
	recs, err := Read(env, t0)
	if err != nil || len(recs) != 1 || recs[0].Session != "s0" {
		t.Fatalf("%+v %v", recs, err)
	}
	if got := strings.Join(files(t, dir), ","); got != before {
		t.Errorf("Read changed the folder: %v -> %v", before, got)
	}
}

func TestReadSkipsLinkToRecord(t *testing.T) {
	env, dir := newEnv(t)
	Write(env, t0, "working", in("a", "/w"), "/x")
	os.Remove(filepath.Join(dir, files(t, dir)[0]))
	os.WriteFile(filepath.Join(t.TempDir(), "r"), nil, 0o600)
	target := filepath.Join(t.TempDir(), "rec.json")
	os.WriteFile(target, []byte(`{"session":"x","dir":"/w","state":"working","time":"`+t0.Format(time.RFC3339)+`"}`), 0o600)
	os.Symlink(target, filepath.Join(dir, "link.json"))
	if recs, _ := Read(env, t0); len(recs) != 0 {
		t.Fatalf("followed a link: %+v", recs)
	}
}

func TestReadMissingAndUnlistable(t *testing.T) {
	env, dir := newEnv(t)
	if recs, err := Read(env, t0); err != nil || recs != nil {
		t.Fatalf("%v %v", recs, err)
	}
	if recs, err := Read(nil, t0); err != nil || recs != nil {
		t.Fatalf("no home: %v %v", recs, err)
	}
	os.MkdirAll(dir, 0o700)
	os.Chmod(dir, 0)
	defer os.Chmod(dir, 0o700)
	if _, err := Read(env, t0); err == nil {
		t.Error("want error for a folder that cannot be listed")
	}
}
