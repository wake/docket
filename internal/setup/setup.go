// Package setup installs the docket schema and skill into a repository.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/wake/docket/internal/safepath"
)

// FileStatus says what init did with one installed file.
type FileStatus string

const (
	Written     FileStatus = "written"
	Unchanged   FileStatus = "unchanged"
	Differs     FileStatus = "differs"
	Overwritten FileStatus = "overwritten"
)

// FileKind says which part of the installation a file belongs to.
type FileKind string

const (
	KindSchema FileKind = "schema"
	KindSkill  FileKind = "skill"
	KindBlock  FileKind = "block"
)

// File is one installed file; Path is relative to the repository root. Kind
// is for callers and is not part of the JSON output.
type File struct {
	Path   string     `json:"path"`
	Status FileStatus `json:"status"`
	Kind   FileKind   `json:"-"`
}

// Result is what Init reports.
type Result struct {
	OpenSpecInitialized bool   `json:"openspec_initialized"`
	Files               []File `json:"files"`
}

// Assets are the files Init installs.
type Assets struct {
	Schema fs.FS  // installed under schemaDir
	Skill  fs.FS  // installed under each of skillDirs
	Block  string // the Docket block kept in AGENTS.md and CLAUDE.md
}

// ProblemKind says why a path stopped Init.
type ProblemKind string

const (
	ProblemSymlink    ProblemKind = "symlink"
	ProblemUnreadable ProblemKind = "unreadable"
	ProblemMalformed  ProblemKind = "malformed" // Err wraps errMalformed
)

// Problem is one path that failed a check.
type Problem struct {
	Path string // relative to the repository root
	Kind ProblemKind
	Err  error
}

// Refused is returned when a check failed; nothing was written.
type Refused struct{ Problems []Problem }

func (r *Refused) Error() string {
	msg := "init refused:"
	for _, p := range r.Problems {
		msg += fmt.Sprintf(" %s: %v;", p.Path, p.Err)
	}
	return msg
}

// schemaDir is where the schema lives, relative to the repository root.
const schemaDir = "openspec/schemas/docket"

// agentsFile and claudeFile are the files that carry the Docket block.
const (
	agentsFile = "AGENTS.md"
	claudeFile = "CLAUDE.md"
)

// skillDirs are the directories that receive the skill.
var skillDirs = []string{".claude/skills/docket", ".agents/skills/docket"}

// target is one file Init installs.
type target struct {
	rel  string
	want []byte
	kind FileKind
}

// decided is a target with what the check phase settled: the status, the
// bytes to write, and whether the file existed.
type decided struct {
	target
	status  FileStatus
	data    []byte
	existed bool
	// viaAgents marks a CLAUDE.md that already leads to AGENTS.md (an import
	// or a link); Inspect leaves it out and Init ignores it.
	viaAgents bool
}

// Init installs the schema and the skill under root. It first decides the
// status of every file and checks every path; only when all checks pass does
// it call initOpenSpec (when <root>/openspec is missing) and write. A refused
// run writes nothing and returns *Refused. A differing file is left alone
// (Differs) unless force is set (Overwritten).
func Init(root string, a Assets, force bool, initOpenSpec func() error) (Result, error) {
	res := Result{Files: []File{}}
	plan, problems, err := decideAll(root, a, force)
	if err != nil {
		return res, err
	}
	if len(problems) > 0 {
		return res, &Refused{Problems: problems}
	}
	if _, err := os.Lstat(filepath.Join(root, "openspec")); errors.Is(err, fs.ErrNotExist) {
		if err := initOpenSpec(); err != nil {
			return res, err
		}
		res.OpenSpecInitialized = true
	} else if err != nil {
		return res, err
	}
	for _, d := range plan {
		if err := write(root, d); err != nil {
			return res, err
		}
		res.Files = append(res.Files, File{Path: d.rel, Status: d.status, Kind: d.kind})
	}
	return res, nil
}

// decideAll checks the openspec path and settles every target. It returns the
// decisions that passed and the problems that would make Init refuse; an
// unsafe openspec path ends the check at once. It changes nothing.
func decideAll(root string, a Assets, force bool) ([]decided, []Problem, error) {
	if err := safepath.Check(root, "openspec"); err != nil {
		return nil, []Problem{problemFor("openspec", err)}, nil
	}
	targets, err := targetsOf(a)
	if err != nil {
		return nil, nil, err
	}
	var problems []Problem
	plan := make([]decided, 0, len(targets))
	for _, t := range targets {
		d, p := decide(root, t, a.Block, force)
		if p != nil {
			problems = append(problems, *p)
			continue
		}
		plan = append(plan, d)
	}
	return plan, problems, nil
}

// Inspect reports what Init without force would do under root, writing and
// starting nothing: the files it would install or keep with Init's statuses,
// and the problems that would make it refuse. Unlike Init it still returns
// the files that passed. A CLAUDE.md that imports AGENTS.md, or links to it,
// is left out. The error is for failing to read the assets.
func Inspect(root string, a Assets) ([]File, []Problem, error) {
	plan, problems, err := decideAll(root, a, false)
	if err != nil {
		return nil, nil, err
	}
	files := make([]File, 0, len(plan))
	for _, d := range plan {
		if d.viaAgents {
			continue
		}
		files = append(files, File{Path: d.rel, Status: d.status, Kind: d.kind})
	}
	return files, problems, nil
}

// targetsOf lists every file to install: the schema, the skill in each of
// skillDirs, then AGENTS.md and CLAUDE.md (whose bytes are decided later).
func targetsOf(a Assets) ([]target, error) {
	var out []target
	add := func(src fs.FS, dir string, kind FileKind) error {
		return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			want, err := fs.ReadFile(src, p)
			if err != nil {
				return err
			}
			out = append(out, target{rel: path.Join(dir, p), want: want, kind: kind})
			return nil
		})
	}
	if err := add(a.Schema, schemaDir, KindSchema); err != nil {
		return nil, err
	}
	for _, dir := range skillDirs {
		if err := add(a.Skill, dir, KindSkill); err != nil {
			return nil, err
		}
	}
	out = append(out, target{rel: agentsFile, kind: KindBlock}, target{rel: claudeFile, kind: KindBlock})
	return out, nil
}

// problemFor classifies a safepath.Check error.
func problemFor(rel string, err error) Problem {
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		return Problem{Path: rel, Kind: ProblemSymlink, Err: err}
	}
	return Problem{Path: rel, Kind: ProblemUnreadable, Err: err}
}

// decide checks t's path and reads the file there to say what a write would
// do. It changes nothing.
func decide(root string, t target, block string, force bool) (decided, *Problem) {
	d := decided{target: t}
	abs := filepath.Join(root, filepath.FromSlash(t.rel))
	if t.rel == claudeFile {
		if tgt, err := os.Readlink(abs); err == nil && filepath.Clean(tgt) == agentsFile {
			d.status, d.viaAgents = Unchanged, true
			return d, nil
		}
	}
	if err := safepath.Check(root, t.rel); err != nil {
		p := problemFor(t.rel, err)
		return d, &p
	}
	got, err := readNoFollow(abs)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d, &Problem{Path: t.rel, Kind: ProblemUnreadable, Err: err}
	}
	d.existed = err == nil
	if t.kind == KindBlock {
		return decideBlock(d, got, block, force)
	}
	d.data = t.want
	switch {
	case !d.existed:
		d.status = Written
	case bytes.Equal(got, t.want):
		d.status = Unchanged
	case !force:
		d.status = Differs
	default:
		d.status = Overwritten
	}
	return d, nil
}

// decideBlock settles an AGENTS.md or CLAUDE.md target; got is the current
// content (nil when the file is missing).
func decideBlock(d decided, got []byte, block string, force bool) (decided, *Problem) {
	switch {
	case d.rel == claudeFile && !d.existed:
		d.status, d.data = Written, []byte("@AGENTS.md\n")
		return d, nil
	case d.rel == claudeFile && importsAgents(got):
		// Markers still have to form a block or none at all.
		if _, _, _, err := FindBlock(got); err != nil {
			return d, &Problem{Path: d.rel, Kind: ProblemMalformed, Err: err}
		}
		d.status, d.viaAgents = Unchanged, true
		return d, nil
	}
	next, status, err := placeBlock(got, block, force)
	if err != nil {
		return d, &Problem{Path: d.rel, Kind: ProblemMalformed, Err: err}
	}
	d.status, d.data = status, next
	return d, nil
}

// write carries out a decision, never following a link at the destination.
func write(root string, d decided) error {
	if d.status == Unchanged || d.status == Differs {
		return nil
	}
	dst := filepath.Join(root, filepath.FromSlash(d.rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", d.rel, err)
	}
	if d.existed {
		info, err := os.Lstat(dst)
		if err != nil {
			return fmt.Errorf("write %s: %w", d.rel, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("write %s: not a regular file", d.rel)
		}
		return install(dst, d.data, info.Mode().Perm(), true, d.rel)
	}
	return install(dst, d.data, 0o644, false, d.rel)
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
