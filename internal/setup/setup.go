// Package setup decides and writes the files dkt init installs.
package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/wake/docket/internal/safefile"
	"github.com/wake/docket/internal/safepath"
)

type Kind string

const (
	KindSchema   Kind = "schema"
	KindConfig   Kind = "config"
	KindSkill    Kind = "skill"
	KindOpenSpec Kind = "openspec"
	KindWorkflow Kind = "workflow"
)

type Status string

const (
	StatusSame     Status = "same"
	StatusMissing  Status = "missing"
	StatusOutdated Status = "outdated"
	StatusDiffers  Status = "differs"
)

type Action string

const (
	ActionNone      Action = "none"
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionOverwrite Action = "overwrite"
	ActionKeep      Action = "keep"
)

// Decision is one target: Path is slash-separated and relative to the repository root.
type Decision struct {
	Path   string
	Kind   Kind
	Status Status
	Action Action

	// Config is set for the config kind only.
	Config *ConfigState

	// Workflow is set for the workflow kind when the file exists.
	Workflow *WorkflowState

	data    []byte
	existed bool
}

type ProblemKind string

const (
	ProblemSymlink    ProblemKind = "symlink"
	ProblemUnreadable ProblemKind = "unreadable"
)

// Problem is one path that failed a check. For a link, Path is the link itself.
type Problem struct {
	Path string
	Kind ProblemKind
	Err  error
}

type target struct {
	rel     string
	want    []byte
	kind    Kind
	version int // the shipped template version, for the workflow kind
}

// Decide reads every target under root, changing nothing. shipped is the
// shipped/ tree. force turns keep into overwrite. A target that fails a check
// is left out of the decisions and reported as a Problem; the error is for
// failing to read shipped.
func Decide(root string, shipped fs.FS, force bool) ([]Decision, []Problem, error) {
	targets, err := targetsOf(shipped)
	if err != nil {
		return nil, nil, err
	}
	var ds []Decision
	var problems []Problem
	seen := map[string]bool{} // one problem per link
	for _, t := range targets {
		d, p := decide(root, t, force)
		if p != nil {
			if p.Kind != ProblemSymlink || !seen[p.Path] {
				problems = append(problems, *p)
			}
			seen[p.Path] = true
			continue
		}
		ds = append(ds, d)
	}
	return ds, problems, nil
}

// targetsOf lists the schema files sorted by path, then openspec/config.yaml, then the skill in each skill root, then the workflow file.
func targetsOf(shipped fs.FS) ([]target, error) {
	var out []target
	err := fs.WalkDir(shipped, "schema", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want, err := fs.ReadFile(shipped, p)
		if err != nil {
			return err
		}
		out = append(out, target{rel: path.Join("openspec/schemas/docket", p[len("schema/"):]), want: want, kind: KindSchema})
		return nil
	})
	if err != nil {
		return nil, err
	}
	ops, err := readShippedOps(shipped)
	if err != nil {
		return nil, err
	}
	out = append(out, target{rel: configPath, want: ops, kind: KindConfig})
	skill, err := fs.ReadFile(shipped, "skill/SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, r := range []string{".claude/skills/docket/SKILL.md", ".agents/skills/docket/SKILL.md"} {
		out = append(out, target{rel: r, want: skill, kind: KindSkill})
	}
	tpl, err := fs.ReadFile(shipped, "workflow/workflow.md")
	if err != nil {
		return nil, err
	}
	v := templateVersion(tpl)
	if v == 0 {
		return nil, errors.New("the shipped workflow template has no template version")
	}
	out = append(out, target{rel: workflowPath, want: tpl, kind: KindWorkflow, version: v})
	return out, nil
}

func problemFor(rel string, err error) Problem {
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		return Problem{Path: sl.Path, Kind: ProblemSymlink, Err: err}
	}
	return Problem{Path: rel, Kind: ProblemUnreadable, Err: err}
}

func decide(root string, t target, force bool) (Decision, *Problem) {
	d := Decision{Path: t.rel, Kind: t.kind, data: t.want}
	if err := safepath.Check(root, t.rel); err != nil {
		p := problemFor(t.rel, err)
		return d, &p
	}
	abs := filepath.Join(root, filepath.FromSlash(t.rel))
	var got []byte
	fi, err := os.Lstat(abs)
	if err == nil && !fi.Mode().IsRegular() { // never open a FIFO or socket
		err = fmt.Errorf("%s is not a regular file", t.rel)
	} else if err == nil {
		got, err = safefile.ReadNoFollow(abs)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d, &Problem{Path: t.rel, Kind: ProblemUnreadable, Err: err}
	}
	d.existed = err == nil
	if t.kind == KindConfig {
		return decideConfig(d, got, t.want)
	}
	if t.kind == KindWorkflow {
		return decideWorkflow(d, got, t.want, t.version), nil
	}
	sum := sha256.Sum256(got)
	switch {
	case !d.existed:
		d.Status, d.Action = StatusMissing, ActionCreate
	case bytes.Equal(got, t.want):
		d.Status, d.Action = StatusSame, ActionNone
	case slices.Contains(Earlier[t.rel], hex.EncodeToString(sum[:])):
		d.Status, d.Action = StatusOutdated, ActionUpdate
	case force:
		d.Status, d.Action = StatusDiffers, ActionOverwrite
	default:
		d.Status, d.Action = StatusDiffers, ActionKeep
	}
	return d, nil
}

// Write carries out the decisions whose Action is create, update or overwrite,
// never following a link, and returns the first error. The decisions must
// come from a Decide run just before: Write re-checks nothing beyond not
// following the final link. Every write goes through root, which a folder
// swapped for a link after Decide cannot lead out of.
func Write(root string, ds []Decision) error {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer dir.Close()
	for _, d := range ds {
		if d.Action != ActionCreate && d.Action != ActionUpdate && d.Action != ActionOverwrite {
			continue
		}
		if err := write(dir, d); err != nil {
			return err
		}
	}
	return nil
}

func write(root *os.Root, d Decision) error {
	name := filepath.FromSlash(d.Path)
	if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", d.Path, err)
	}
	if d.existed {
		info, err := root.Lstat(name)
		if err != nil {
			return fmt.Errorf("write %s: %w", d.Path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("write %s: not a regular file", d.Path)
		}
		return safefile.InstallIn(root, name, d.data, info.Mode().Perm(), true, d.Path)
	}
	return safefile.InstallIn(root, name, d.data, 0o644, false, d.Path)
}
