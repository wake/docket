package openspec_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/testutil"
)

var (
	changeArgs = []string{"show", "add-x", "--type", "change", "--json", "--deltas-only"}
	specArgs   = []string{"show", "booking", "--type", "spec", "--json"}
	validArgs  = []string{"validate", "add-x", "--type", "change", "--json"}
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// jsonRunner returns a Runner whose fake answers args with code and body.
func jsonRunner(t *testing.T, code int, body string, args []string) (*openspec.Runner, string) {
	t.Helper()
	dir := t.TempDir()
	testutil.OpenSpecAnswer(t, dir, code, body, args...)
	return newRunner(t, "1.14.0", "FAKE_OPENSPEC_JSON="+dir)
}

func TestShowChange(t *testing.T) {
	r, _ := jsonRunner(t, 0, fixture(t, "show-change.json"), changeArgs)
	got, err := r.ShowChange("add-x")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("got %d deltas: %+v", len(got), got)
	}
	want := []openspec.Delta{
		{Spec: "booking", Operation: "ADDED", Requirements: []openspec.Requirement{{Name: "Join when full", Scenarios: []openspec.Scenario{{Name: "Join when full — rejected"}}}}},
		{Spec: "booking", Operation: "MODIFIED", Requirements: []openspec.Requirement{{Name: "Leave", Scenarios: []openspec.Scenario{{Name: "Leave anytime"}}}}},
		{Spec: "booking", Operation: "REMOVED", Requirements: []openspec.Requirement{{Name: "Old"}}},
		{Spec: "booking", Operation: "RENAMED", Rename: &openspec.Rename{From: "Join", To: "Join the list"}},
	}
	if !reflect.DeepEqual(got[:4], want) {
		t.Fatalf("got %+v\nwant %+v", got[:4], want)
	}
}

func TestShowSpec(t *testing.T) {
	r, _ := jsonRunner(t, 0, fixture(t, "show-spec.json"), specArgs)
	got, err := r.ShowSpec("booking")
	if err != nil {
		t.Fatal(err)
	}
	want := []openspec.Requirement{
		{Name: "Join", Scenarios: []openspec.Scenario{{Name: "Success"}}},
		{Name: "Leave", Scenarios: []openspec.Scenario{{Name: "Success"}}},
		{Name: "Old", Scenarios: []openspec.Scenario{{Name: "Old one"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateIssues(t *testing.T) {
	r, _ := jsonRunner(t, 1, fixture(t, "validate-invalid.json"), validArgs)
	got, err := r.Validate("add-x")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Level != "ERROR" || got[0].Path != "booking/spec.md" || got[0].Line != 0 ||
		!strings.HasPrefix(got[0].Message, `MODIFIED "Leave" omits`) || got[1].Path != "rooms/spec.md" {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateLine(t *testing.T) {
	body := `{"items":[{"id":"other","issues":[{"level":"ERROR","path":"x","message":"no"}]},{"id":"add-x","issues":[{"level":"WARNING","path":"a/spec.md","message":"m","line":7}]}]}`
	r, _ := jsonRunner(t, 1, body, validArgs)
	got, err := r.Validate("add-x")
	want := []openspec.Issue{{Level: "WARNING", Path: "a/spec.md", Message: "m", Line: 7}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestValidateValid(t *testing.T) {
	r, _ := jsonRunner(t, 0, `{"items":[{"id":"add-x","valid":true,"issues":[]}]}`, validArgs)
	got, err := r.Validate("add-x")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestShowFailedHasMessages(t *testing.T) {
	r, _ := jsonRunner(t, 1, fixture(t, "show-missing.json"), changeArgs)
	_, err := r.ShowChange("add-x")
	var fe *openspec.FailedError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v", err)
	}
	if len(fe.Messages) != 1 || !strings.HasPrefix(fe.Messages[0], `Change "add-zz" not found`) {
		t.Fatalf("messages = %q", fe.Messages)
	}
	if !strings.HasSuffix(fe.Error(), ": "+fe.Messages[0]) {
		t.Fatalf("Error() = %q", fe.Error())
	}
}

func TestValidateNoItemsFails(t *testing.T) {
	r, _ := jsonRunner(t, 1, `{"status":[{"message":"boom"},{"message":"bang"}]}`, validArgs)
	_, err := r.Validate("add-x")
	var fe *openspec.FailedError
	if !errors.As(err, &fe) || !reflect.DeepEqual(fe.Messages, []string{"boom", "bang"}) || !strings.HasSuffix(fe.Error(), ": boom; bang") {
		t.Fatalf("err = %v", err)
	}
	r, _ = jsonRunner(t, 0, `{"x":1}`, validArgs)
	if _, err := r.Validate("add-x"); err == nil || !strings.Contains(err.Error(), "openspec validate") {
		t.Fatalf("err = %v", err)
	}
}

func TestShowUndecodable(t *testing.T) {
	r, _ := jsonRunner(t, 0, "not json", specArgs)
	_, err := r.ShowSpec("booking")
	if err == nil || !strings.Contains(err.Error(), "openspec show booking") {
		t.Fatalf("err = %v", err)
	}
}

func TestShowIgnoresUnknownFields(t *testing.T) {
	r, _ := jsonRunner(t, 0, `{"zzz":1,"requirements":[{"name":"A","extra":true,"scenarios":[{"name":"S","new":1}]}]}`, specArgs)
	got, err := r.ShowSpec("booking")
	if err != nil || len(got) != 1 || got[0].Name != "A" || got[0].Scenarios[0].Name != "S" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestJSONUnsupportedVersion(t *testing.T) {
	r, log := newRunner(t, "1.13.2")
	var ue *openspec.UnsupportedError
	if _, err := r.ShowChange("add-x"); !errors.As(err, &ue) {
		t.Fatalf("err = %v", err)
	}
	if got := readLog(t, log); len(got) != 1 || got[0][1] != "--version" {
		t.Fatalf("log = %q", got)
	}
}
