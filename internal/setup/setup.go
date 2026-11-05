// Package setup decides and writes the files dkt init installs.
package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/wake/docket/internal/safepath"
)

type Kind string

const (
	KindSchema   Kind = "schema"
	KindConfig   Kind = "config"
	KindSkill    Kind = "skill"
	KindOpenSpec Kind = "openspec"
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
	rel  string
	want []byte
	kind Kind
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

// targetsOf lists the schema files sorted by path, then the skill in each skill root.
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
	skill, err := fs.ReadFile(shipped, "skill/SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, r := range []string{".claude/skills/docket/SKILL.md", ".agents/skills/docket/SKILL.md"} {
		out = append(out, target{rel: r, want: skill, kind: KindSkill})
	}
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
	got, err := readNoFollow(abs)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d, &Problem{Path: t.rel, Kind: ProblemUnreadable, Err: err}
	}
	d.existed = err == nil
	sum := sha256.Sum256(got)
	switch {
	case !d.existed:
		d.Status, d.Action = StatusMissing, ActionCreate
	case bytes.Equal(got, t.want):
		d.Status, d.Action = StatusSame, ActionNone
	case slices.Contains(earlier[t.rel], hex.EncodeToString(sum[:])):
		d.Status, d.Action = StatusOutdated, ActionUpdate
	case force:
		d.Status, d.Action = StatusDiffers, ActionOverwrite
	default:
		d.Status, d.Action = StatusDiffers, ActionKeep
	}
	return d, nil
}

// Write carries out the decisions whose Action is create, update or overwrite,
// never following a link, and returns the first error.
func Write(root string, ds []Decision) error {
	for _, d := range ds {
		if d.Action != ActionCreate && d.Action != ActionUpdate && d.Action != ActionOverwrite {
			continue
		}
		if err := write(root, d); err != nil {
			return err
		}
	}
	return nil
}

func write(root string, d Decision) error {
	dst := filepath.Join(root, filepath.FromSlash(d.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", d.Path, err)
	}
	if d.existed {
		info, err := os.Lstat(dst)
		if err != nil {
			return fmt.Errorf("write %s: %w", d.Path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("write %s: not a regular file", d.Path)
		}
		return install(dst, d.data, info.Mode().Perm(), true, d.Path)
	}
	return install(dst, d.data, 0o644, false, d.Path)
}

// rename is os.Rename; tests replace it to make the last step fail.
var rename = os.Rename

// install writes data to a temporary file beside dst and renames it over dst,
// so a failed write leaves dst as it was (or missing). An existing file's
// permission bits are kept exactly (keepMode); a new file gets perm subject
// to the umask.
func install(dst string, data []byte, perm os.FileMode, keepMode bool, rel string) (err error) {
	tmp := fmt.Sprintf("%s.docket-tmp-%d", dst, os.Getpid())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, perm)
	if err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()
	if keepMode {
		err = f.Chmod(perm)
	}
	if err == nil {
		_, err = f.Write(data)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = rename(tmp, dst)
	}
	if err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

// readNoFollow reads the file at p without following a link there.
func readNoFollow(p string) ([]byte, error) {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
