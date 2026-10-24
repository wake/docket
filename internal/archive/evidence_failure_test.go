package archive

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUnmergedSecondChangeStopsBeforeAnyArchive(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Git("checkout", "-q", "-b", "feature")
	c1 := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Commit("work\n\nChange: add-y\nTask: 1", map[string]string{
		"openspec/changes/add-y/.openspec.yaml": "schema: docket\nx-base: " + c1 + "\n",
		"openspec/changes/add-y/plan.md":        archPlan,
	})
	rep, _ := Archive(f.input(e, nil, "add-x", "add-y"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"change-not-merged"}) || rep.Findings[0].Change != "add-y" ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"", "preconditions"}) || len(rep.Archived) != 0 || f.called("archive") {
		t.Fatalf("%+v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.Root, "openspec/changes/add-x/plan.md")); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceWriteFailures(t *testing.T) {
	arch := "openspec/changes/archive/2026-11-02-add-x"
	check := func(t *testing.T, rep ArchiveReport, rule string) {
		t.Helper()
		if !reflect.DeepEqual(ruleList(rep.Findings), []string{rule}) || rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "evidence"}) ||
			len(rep.Archived) != 1 || rep.Archived[0].Change != "add-x" || rep.Archived[0].Folder != arch || rep.Archived[0].Evidence != "" {
			t.Fatalf("%+v", rep)
		}
	}
	t.Run("ordinary error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		_, e, f := archRepo(t, "add-x")
		f.hooks = map[string]func(){"add-x": func() {
			if err := os.Chmod(filepath.Join(e.Root, arch), 0o555); err != nil {
				t.Fatal(err)
			}
		}}
		t.Cleanup(func() { os.Chmod(filepath.Join(e.Root, arch), 0o755) })
		rep, err := Archive(f.input(e, nil, "add-x"))
		if err != nil {
			t.Fatal(err)
		}
		check(t, rep, "evidence-failed")
	})
	t.Run("write fails midway", func(t *testing.T) {
		_, e, f := archRepo(t, "add-x")
		writeTemp = func(f *os.File, b []byte) error {
			_, _ = f.Write(b[:len(b)/2])
			return errors.New("disk full")
		}
		t.Cleanup(func() { writeTemp = writeAll })
		rep, err := Archive(f.input(e, nil, "add-x"))
		if err != nil {
			t.Fatal(err)
		}
		check(t, rep, "evidence-failed")
		if _, err := os.Stat(filepath.Join(e.Root, arch, "evidence.md")); !os.IsNotExist(err) {
			t.Errorf("partial evidence.md left: %v", err)
		}
		noTempFiles(t, filepath.Join(e.Root, arch))
	})
	t.Run("evidence.md already exists", func(t *testing.T) {
		_, e, f := archRepo(t, "add-x")
		f.hooks = map[string]func(){"add-x": func() {
			if err := os.WriteFile(filepath.Join(e.Root, arch, "evidence.md"), []byte("mine\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}}
		rep, err := Archive(f.input(e, nil, "add-x"))
		if err != nil {
			t.Fatal(err)
		}
		check(t, rep, "evidence-failed")
		if b, _ := os.ReadFile(filepath.Join(e.Root, arch, "evidence.md")); string(b) != "mine\n" {
			t.Errorf("evidence.md overwritten: %q", b)
		}
		noTempFiles(t, filepath.Join(e.Root, arch))
	})
	t.Run("evidence.md is a link", func(t *testing.T) {
		_, e, f := archRepo(t, "add-x")
		target := filepath.Join(t.TempDir(), "target.txt")
		if err := os.WriteFile(target, []byte("keep\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		f.hooks = map[string]func(){"add-x": func() {
			if err := os.Symlink(target, filepath.Join(e.Root, arch, "evidence.md")); err != nil {
				t.Fatal(err)
			}
		}}
		rep, err := Archive(f.input(e, nil, "add-x"))
		if err != nil {
			t.Fatal(err)
		}
		check(t, rep, "symlink-refused")
		if b, _ := os.ReadFile(target); string(b) != "keep\n" {
			t.Fatalf("target changed: %q", b)
		}
	})
}

func TestEvidenceOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"":                                       "none",
		"https://TOKEN@host/x":                   "https://host/x",
		"https://user:pass@host/x":               "https://host/x",
		"http://user:pass@host/x":                "http://host/x",
		"https://host/x":                         "https://host/x",
		"ssh://git:secret@host/x":                "ssh://git@host/x",
		"ssh://git@host/x":                       "ssh://git@host/x",
		"git://host/x":                           "git://host/x",
		"file:///srv/repo":                       "none",
		"/srv/repo":                              "none",
		"./repo":                                 "none",
		"../repo":                                "none",
		"~/repo":                                 "none",
		"repo":                                   "none",
		"git@host:org/repo.git":                  "git@host:org/repo.git",
		"host:org/repo.git":                      "none",
		"git@github.com:wake/docket.git":         "git@github.com:wake/docket.git",
		"https://host/x?access_token=SECRET":     "https://host/x",
		"https://host/x#access_token=SECRET":     "https://host/x",
		"https://host:8443/a/b?x=1#y":            "https://host:8443/a/b",
		"ssh://git@host/x?token=SECRET":          "ssh://git@host/x",
		"ssh://git@host/x#frag":                  "ssh://git@host/x",
		"git+ssh://git:pw@host/x?token=SECRET#f": "git+ssh://git@host/x",
		"ftp://host/x":                           "none",
		"C:/Users/alice/repo":                    "none",
		`C:\Users\alice\repo`:                    "none",
		"git@host:org/repo?token=SECRET":         "none",
		"git@host:org/repo#frag":                 "none",
		"https:///x":                             "none",
	} {
		if got := evidenceOrigin(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
