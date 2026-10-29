package archive

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const titleFolder = "openspec/changes/archive/2026-11-02-add-x"

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// titleStopped checks the outcome every failing title step shares.
func titleStopped(t *testing.T, root string, rep ArchiveReport) {
	t.Helper()
	if rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "spec-title"}) || len(rep.Archived) != 1 ||
		rep.Archived[0].Folder != titleFolder || rep.Archived[0].Evidence != "" {
		t.Fatalf("%+v", rep)
	}
	if _, err := os.Stat(filepath.Join(root, titleFolder)); err != nil {
		t.Fatal("archived folder is gone")
	}
	if _, err := os.Stat(filepath.Join(root, titleFolder, "evidence.md")); err == nil {
		t.Fatal("evidence.md was written")
	}
}

func runTitles(t *testing.T, files map[string]string) (*testutil.Repo, ArchiveReport) {
	t.Helper()
	r, e, f := archRepo(t, "add-x")
	if len(files) > 0 {
		r.Commit("titles", files)
	}
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	return r, rep
}

func TestTitleCarriedToNewSpec(t *testing.T) {
	r, rep := runTitles(t, map[string]string{"openspec/changes/add-x/specs/booking/docket.yaml": "title: Waitlist\n"})
	if rep.Stopped != nil || rep.Archived[0].Evidence == "" {
		t.Fatalf("%+v", rep)
	}
	if got := readRepoFile(t, r.Dir, "openspec/specs/booking/docket.yaml"); got != "title: Waitlist\n" {
		t.Fatalf("%q", got)
	}
}

func TestTitleReplacedOnExistingSpec(t *testing.T) {
	r, rep := runTitles(t, map[string]string{
		"openspec/specs/booking/docket.yaml":               "title: Bookings\nowner: x\n",
		"openspec/changes/add-x/specs/booking/docket.yaml": "title: Booking\n",
	})
	if rep.Stopped != nil {
		t.Fatalf("%+v", rep)
	}
	if got := readRepoFile(t, r.Dir, "openspec/specs/booking/docket.yaml"); got != "title: Booking\nowner: x\n" {
		t.Fatalf("%q", got)
	}
}

func TestNoDeltaTitleLeavesSpecAlone(t *testing.T) {
	r, rep := runTitles(t, map[string]string{"openspec/specs/booking/docket.yaml": "title: Bookings\nowner: x\n"})
	if rep.Stopped != nil {
		t.Fatalf("%+v", rep)
	}
	if got := readRepoFile(t, r.Dir, "openspec/specs/booking/docket.yaml"); got != "title: Bookings\nowner: x\n" {
		t.Fatalf("%q", got)
	}
	r2, rep2 := runTitles(t, nil)
	if rep2.Stopped != nil {
		t.Fatalf("%+v", rep2)
	}
	if _, err := os.Stat(filepath.Join(r2.Dir, "openspec/specs/booking/docket.yaml")); err == nil {
		t.Fatal("created")
	}
}

func TestTitleNestedCapability(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Commit("nested", map[string]string{
		"openspec/changes/add-x/specs/booking/waitlist/spec.md":     modDelta("Cancel booking"),
		"openspec/changes/add-x/specs/booking/waitlist/docket.yaml": "title: Waitlist\n",
	})
	cancel := `"operation":"MODIFIED","requirement":{"name":"Cancel booking","scenarios":[{"name":"ok"}]}}`
	f.show = `{"deltas":[{"spec":"booking",` + cancel + `,{"spec":"booking/waitlist",` + cancel + `]}`
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || rep.Stopped != nil {
		t.Fatalf("%v %+v", err, rep)
	}
	if got := readRepoFile(t, r.Dir, "openspec/specs/booking/waitlist/docket.yaml"); got != "title: Waitlist\n" {
		t.Fatalf("%q", got)
	}
}

func TestTitleTargetIsLink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outside, []byte("title: Out\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, e, f := archRepo(t, "add-x")
	if err := os.MkdirAll(filepath.Join(r.Dir, "openspec/specs/booking"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(r.Dir, "openspec/specs/booking/docket.yaml")); err != nil {
		t.Fatal(err)
	}
	r.Commit("titles", map[string]string{"openspec/changes/add-x/specs/booking/docket.yaml": "title: Booking\n"})
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	titleStopped(t, r.Dir, rep)
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "spec-title-failed" {
		t.Fatalf("%+v", rep.Findings)
	}
	if b, _ := os.ReadFile(outside); string(b) != "title: Out\n" {
		t.Fatalf("%q", b)
	}
}

func TestTitleDeltaInvalid(t *testing.T) {
	for _, body := range []string{"- a\n", "title: \"  \"\n"} {
		r, rep := runTitles(t, map[string]string{
			"openspec/specs/booking/docket.yaml":               "title: Bookings\n",
			"openspec/changes/add-x/specs/booking/docket.yaml": body,
		})
		titleStopped(t, r.Dir, rep)
		if len(rep.Findings) != 1 || rep.Findings[0].Rule != "spec-title-failed" || !strings.Contains(rep.Findings[0].Message, "booking") {
			t.Fatalf("%q %+v", body, rep.Findings)
		}
		if got := readRepoFile(t, r.Dir, "openspec/specs/booking/docket.yaml"); got != "title: Bookings\n" {
			t.Fatalf("%q", got)
		}
	}
}

func TestTitleWriteFails(t *testing.T) {
	writeTemp = func(f *os.File, b []byte) error { return errors.New("disk full") }
	t.Cleanup(func() { writeTemp = writeAll })
	r, rep := runTitles(t, map[string]string{
		"openspec/specs/booking/docket.yaml":               "title: Bookings\n",
		"openspec/changes/add-x/specs/booking/docket.yaml": "title: Booking\n",
	})
	titleStopped(t, r.Dir, rep)
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "spec-title-failed" || !strings.Contains(rep.Findings[0].Message, "disk full") {
		t.Fatalf("%+v", rep.Findings)
	}
	if got := readRepoFile(t, r.Dir, "openspec/specs/booking/docket.yaml"); got != "title: Bookings\n" {
		t.Fatalf("%q", got)
	}
}

func TestTitleTargetUnreadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads anything")
	}
	r, e, f := archRepo(t, "add-x")
	r.Commit("titles", map[string]string{
		"openspec/specs/booking/docket.yaml":               "title: Bookings\n",
		"openspec/changes/add-x/specs/booking/docket.yaml": "title: Booking\n",
	})
	target := filepath.Join(r.Dir, "openspec/specs/booking/docket.yaml")
	f.hooks = map[string]func(){"add-x": func() {
		if err := os.Chmod(target, 0); err != nil {
			t.Fatal(err)
		}
	}}
	t.Cleanup(func() { _ = os.Chmod(target, 0o644) })
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	titleStopped(t, r.Dir, rep)
	if len(rep.Findings) != 0 || len(rep.Unreadable) != 1 || rep.Unreadable[0].Source != "openspec/specs/booking/docket.yaml" {
		t.Fatalf("%+v %+v", rep.Findings, rep.Unreadable)
	}
}
