// Package setup installs the docket schema and skills into a repository.
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
	Removed     FileStatus = "removed"
)

// FileKind says which part of the installation a file belongs to.
type FileKind string

const (
	KindSchema FileKind = "schema"
	KindSkill  FileKind = "skill"
	KindBlock  FileKind = "block"
	KindConfig FileKind = "config"
)

// File is one installed file; Path is relative to the repository root, or
// absolute for a file in the per-machine schema directory. Kind is for
// callers and is not part of the JSON output.
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

// Skill is one skill folder: its name, and its files with paths relative to
// that folder (SKILL.md, ...).
type Skill struct {
	Name  string
	Files fs.FS
}

// Assets are the files Init installs.
type Assets struct {
	Schema fs.FS   // installed under schemaDir
	Skills []Skill // each installed under <root>/<name> for each of skillRoots
	Block  string  // the Docket block kept in AGENTS.md and CLAUDE.md
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

// schemaName is the folder of the schema under the per-machine schemas
// directory.
const schemaName = "docket"

// agentsFile and claudeFile are the files that carry the Docket block.
const (
	agentsFile = "AGENTS.md"
	claudeFile = "CLAUDE.md"
)

// skillRoots are the directories that receive each skill, in a folder named
// after it.
var skillRoots = []string{".claude/skills", ".agents/skills"}

// target is one file Init installs.
type target struct {
	rel  string
	want []byte
	kind FileKind
	// base is the directory rel is under; empty means the repository root.
	// The per-machine schema files have base <schemas dir> and rel
	// docket/<file>.
	base string
}

// shown is the path File reports: rel, or the absolute path under base.
func (t target) shown() string {
	if t.base == "" {
		return t.rel
	}
	return filepath.Join(t.base, filepath.FromSlash(t.rel))
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
	// config is what a config.yaml target found.
	config ConfigSchema
}

// Init installs the schema into schemaDir (the per-machine directory) and the
// skills under root. It first decides the
// status of every file and checks every path; only when all checks pass does
// it call initOpenSpec (when <root>/openspec is missing) and write. A refused
// run writes nothing and returns *Refused. A differing file is left alone
// (Differs) unless force is set (Overwritten).
func Init(root, schemaDir string, a Assets, force bool, initOpenSpec func() error) (Result, error) {
	res := Result{Files: []File{}}
	if err := safepath.Check(root, "openspec"); err != nil {
		return res, &Refused{Problems: []Problem{problemFor("openspec", err)}}
	}
	_, statErr := os.Lstat(filepath.Join(root, "openspec"))
	plan, problems, err := decideAll(root, schemaDir, a, force, statErr == nil)
	if err != nil {
		return res, err
	}
	if len(problems) > 0 {
		return res, &Refused{Problems: problems}
	}
	if errors.Is(statErr, fs.ErrNotExist) {
		if err := initOpenSpec(); err != nil {
			return res, err
		}
		res.OpenSpecInitialized = true
		// OpenSpec may have written config.yaml; decide it now.
		d, p := decide(root, target{rel: configFile, kind: KindConfig}, a.Block, force)
		if p != nil {
			return res, &Refused{Problems: []Problem{*p}}
		}
		plan = append(plan, d)
	} else if statErr != nil {
		return res, statErr
	}
	for _, d := range plan {
		if err := write(root, d); err != nil {
			return res, err
		}
		res.Files = append(res.Files, File{Path: d.shown(), Status: d.status, Kind: d.kind})
	}
	return res, nil
}

// decideAll settles every target. It returns the decisions that passed and
// the problems that would make Init refuse. It does not check the openspec
// directory itself: Init does that first, and Inspect adds it to the problems
// it reports. It changes nothing.
func decideAll(root, schemaDir string, a Assets, force, withConfig bool) ([]decided, []Problem, error) {
	targets, err := targetsOf(a, schemaDir, withConfig)
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

// Report is what Inspect finds.
type Report struct {
	Files    []File
	Problems []Problem
	// Config is what openspec/config.yaml names; empty when it could not be
	// read.
	Config ConfigSchema
}

// Inspect reports what Init without force would do under root, writing and
// starting nothing: the files it would install or keep with Init's statuses,
// and the problems that would make it refuse. Unlike Init it still returns
// the files that passed. A CLAUDE.md that imports AGENTS.md, or links to it,
// is left out. An unsafe openspec path is one of the problems, and the other
// files are still decided. An empty schemaDir leaves the per-machine files
// out. The error is for failing to read the assets.
func Inspect(root, schemaDir string, a Assets) (Report, error) {
	plan, problems, err := decideAll(root, schemaDir, a, false, true)
	if err != nil {
		return Report{}, err
	}
	if err := safepath.Check(root, "openspec"); err != nil {
		problems = append([]Problem{problemFor("openspec", err)}, problems...)
	}
	rep := Report{Files: make([]File, 0, len(plan)), Problems: problems}
	for _, d := range plan {
		if d.kind == KindConfig {
			rep.Config = d.config
		}
		if d.viaAgents {
			continue
		}
		rep.Files = append(rep.Files, File{Path: d.shown(), Status: d.status, Kind: d.kind})
	}
	return rep, nil
}

// targetsOf lists every file to install: the schema in schemaDir (none when
// it is empty), config.yaml when withConfig is set, the skills in each of
// skillRoots, then AGENTS.md and CLAUDE.md (whose bytes are decided later).
func targetsOf(a Assets, schemaDir string, withConfig bool) ([]target, error) {
	var out []target
	add := func(src fs.FS, dir string, kind FileKind, base string) error {
		return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			want, err := fs.ReadFile(src, p)
			if err != nil {
				return err
			}
			out = append(out, target{rel: path.Join(dir, p), want: want, kind: kind, base: base})
			return nil
		})
	}
	if schemaDir != "" {
		if err := add(a.Schema, schemaName, KindSchema, filepath.Dir(schemaDir)); err != nil {
			return nil, err
		}
	}
	if withConfig {
		out = append(out, target{rel: configFile, kind: KindConfig})
	}
	for _, root := range skillRoots {
		for _, sk := range a.Skills {
			if err := add(sk.Files, path.Join(root, sk.Name), KindSkill, ""); err != nil {
				return nil, err
			}
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

// problemForTarget is problemFor for a target under its own base: the paths
// it reports are the ones File reports.
func problemForTarget(t target, err error) Problem {
	var sl *safepath.SymlinkError
	if t.base != "" && errors.As(err, &sl) {
		err = &safepath.SymlinkError{Path: filepath.Join(t.base, filepath.FromSlash(sl.Path))}
	}
	return problemFor(t.shown(), err)
}

// decide checks t's path and reads the file there to say what a write would
// do. It changes nothing.
func decide(root string, t target, block string, force bool) (decided, *Problem) {
	d := decided{target: t}
	if t.base != "" {
		root = t.base
	}
	abs := filepath.Join(root, filepath.FromSlash(t.rel))
	if t.rel == claudeFile {
		if tgt, err := os.Readlink(abs); err == nil && filepath.Clean(tgt) == agentsFile {
			d.status, d.viaAgents = Unchanged, true
			return d, nil
		}
	}
	if err := safepath.Check(root, t.rel); err != nil {
		p := problemForTarget(t, err)
		return d, &p
	}
	got, err := readNoFollow(abs)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return d, &Problem{Path: t.shown(), Kind: ProblemUnreadable, Err: err}
	}
	d.existed = err == nil
	if t.kind == KindConfig {
		return decideConfig(d, got)
	}
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
	if d.base != "" {
		root = d.base
	}
	dst := filepath.Join(root, filepath.FromSlash(d.rel))
	shown := d.shown()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", shown, err)
	}
	if d.existed {
		info, err := os.Lstat(dst)
		if err != nil {
			return fmt.Errorf("write %s: %w", shown, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("write %s: not a regular file", shown)
		}
		return install(dst, d.data, info.Mode().Perm(), true, shown)
	}
	return install(dst, d.data, 0o644, false, shown)
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
