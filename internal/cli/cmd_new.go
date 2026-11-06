package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

type newResult struct {
	Slug      string   `json:"slug"`
	Requested string   `json:"requested"`
	Path      string   `json:"path"` // openspec/changes/<slug>
	Title     string   `json:"title"`
	Kind      string   `json:"kind"`
	Source    string   `json:"source"`
	Used      []newUse `json:"used"` // never null
}

type newUse struct {
	Path    string   `json:"path"`
	Sources []string `json:"sources"`
}

func finding(rule, msg string) (Outcome, error) {
	return Outcome{Findings: []Finding{{Rule: rule, Message: msg}}}, nil
}

// refuse turns a symbolic link on a checked path into symlink-refused; other errors pass through.
func refuse(err error) (Outcome, error) {
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		return finding("symlink-refused", err.Error())
	}
	return Outcome{}, err
}

// lockOpenSpec takes an exclusive lock on <root>/openspec, blocking until it
// is free, so two runs of dkt new cannot interleave. The returned function
// releases it.
func lockOpenSpec(root string) (func(), error) {
	f, err := os.Open(filepath.Join(root, "openspec"))
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// runNew is the body of dkt new.
func runNew(c *Context, slug, kind, source, title string) (Outcome, error) {
	root := c.Repo.Root
	switch {
	case kind == "":
		return Outcome{}, UsageError{Msg: "new needs --kind (" + strings.Join(change.Kinds, ", ") + ")"}
	case !slices.Contains(change.Kinds, kind):
		return Outcome{}, UsageError{Msg: fmt.Sprintf("unknown kind %q; use one of %s", kind, strings.Join(change.Kinds, ", "))}
	case !change.ValidLine(source):
		return Outcome{}, UsageError{Msg: "new needs --source: one line of text that is not blank"}
	case !change.ValidLine(title):
		return Outcome{}, UsageError{Msg: "new needs --title: one line of text that is not blank"}
	}
	if msg, tooLong := change.TitleTooLong(title); tooLong {
		return Outcome{}, UsageError{Msg: "new needs --title: " + msg}
	}
	if err := change.CheckSlug(slug); err != nil {
		return finding("slug-invalid", err.Error())
	}
	proposal := func(s string) string { return "openspec/changes/" + s + "/proposal.md" }
	// Only a link stops here: a stray file at the requested slug is skipped by the slug loop.
	var sl *safepath.SymlinkError
	if err := safepath.Check(root, proposal(slug)); errors.As(err, &sl) {
		return refuse(err)
	}
	if fi, err := os.Lstat(filepath.Join(root, "openspec/schemas/docket/schema.yaml")); err != nil || !fi.Mode().IsRegular() {
		return finding("schema-missing", "the docket schema is not installed (openspec/schemas/docket/schema.yaml); run dkt init")
	}
	r, err := openSpecRunner(c)
	if err != nil {
		return Outcome{}, err
	}
	unlock, err := lockOpenSpec(root)
	if err != nil {
		return Outcome{}, err
	}
	defer unlock()
	inv, err := scan.Changes(c.Git)
	if err != nil {
		return Outcome{}, err
	}
	if len(inv.Unreadable) > 0 {
		var out Outcome
		for _, u := range inv.Unreadable {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
		}
		return out, nil
	}

	var chosen string
	seen := false // the requested slug was found by the scan
	for n := 1; chosen == ""; n++ {
		cand := change.Suffixed(slug, n)
		if slices.ContainsFunc(inv.Uses, func(u scan.Use) bool { return u.Slug == cand }) {
			seen = seen || n == 1
			continue
		}
		_, err := os.Lstat(filepath.Join(root, "openspec/changes", cand))
		if err == nil {
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return Outcome{}, err
		}
		chosen = cand
	}

	used := usedPlaces(inv.Uses, slug)
	if chosen != slug && !seen {
		used = append(used, newUse{Path: "openspec/changes/" + slug, Sources: []string{root}})
	}

	if err := safepath.Check(root, proposal(chosen)); err != nil {
		return refuse(err)
	}
	dir := filepath.Join(root, "openspec/changes", chosen)
	// The lock is held and the folder did not exist, so what is there now was
	// made by this run, unless a link appeared on its path.
	// It removes only what OpenSpec is known to write (and proposal.md when
	// this run created it), so a file another process added is never deleted.
	cleanup := func(err error, ownProposal bool) error {
		if lerr := safepath.Check(root, "openspec/changes/"+chosen); lerr != nil {
			return errors.Join(err, fmt.Errorf("left in place: %w", lerr))
		}
		names := []string{".openspec.yaml", ""}
		if ownProposal {
			names = []string{"proposal.md", ".openspec.yaml", ""}
		}
		for _, n := range names {
			rerr := os.Remove(filepath.Join(dir, n))
			switch {
			case rerr == nil || errors.Is(rerr, fs.ErrNotExist):
			case n == "" && errors.Is(rerr, syscall.ENOTEMPTY):
				return errors.Join(err, errors.New("left in place: openspec/changes/"+chosen+" holds files dkt new did not expect"))
			default:
				return errors.Join(err, fmt.Errorf("left in place: %w", rerr))
			}
		}
		return err
	}
	if _, err := r.Run("new", "change", chosen, "--schema", "docket"); err != nil {
		err = cleanup(err, false)
		var fe *openspec.FailedError
		if errors.As(err, &fe) {
			return finding("openspec-failed", err.Error())
		}
		return Outcome{}, err
	}
	if err := safepath.Check(root, proposal(chosen)); err != nil {
		// A link appeared while OpenSpec ran; removing now could follow it out.
		return refuse(err)
	}
	f, err := os.OpenFile(filepath.Join(root, proposal(chosen)), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return Outcome{}, cleanup(err, false)
	}
	_, err = f.Write(change.FrontMatter(title, kind, source))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return Outcome{}, cleanup(err, true)
	}

	res := newResult{Slug: chosen, Requested: slug, Path: "openspec/changes/" + chosen, Title: title, Kind: kind, Source: source, Used: used}
	human := usedLines(slug, used)
	human = append(human, "created "+res.Path)
	return Outcome{Result: res, Human: human}, nil
}

// usedPlaces groups the uses of slug by path, each with its sources; never nil.
func usedPlaces(uses []scan.Use, slug string) []newUse {
	used := []newUse{}
	for _, u := range uses {
		if u.Slug != slug {
			continue
		}
		i := slices.IndexFunc(used, func(x newUse) bool { return x.Path == u.Path })
		if i < 0 {
			used = append(used, newUse{Path: u.Path})
			i = len(used) - 1
		}
		used[i].Sources = append(used[i].Sources, u.Source)
	}
	return used
}

// usedLines is the human line for each place slug is used.
func usedLines(slug string, used []newUse) []string {
	var human []string
	for _, u := range used {
		s := u.Sources
		list := strings.Join(s, ", ")
		if len(s) > 3 {
			list = fmt.Sprintf("%s and %d more", strings.Join(s[:3], ", "), len(s)-3)
		}
		human = append(human, fmt.Sprintf("%s is used at %s in %s", slug, u.Path, list))
	}
	return human
}
