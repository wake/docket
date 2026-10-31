package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/safepath"
)

// CopyState says what openspec/schemas/docket in the repository is.
type CopyState string

const (
	NoCopy      CopyState = "none"    // no schema.yaml there
	ShippedCopy CopyState = "shipped" // identical to a schema Docket shipped
	EditedCopy  CopyState = "edited"  // anything else
)

// projectCopy is the project's copy of the schema, relative to the root.
const projectCopy = "openspec/schemas/docket"

// removeAll is os.RemoveAll; tests replace it to make the clean-up fail.
var removeAll = os.RemoveAll

// decideCopy says what the project copy is. A link on the folder or under it
// is a problem, as is a file that cannot be read. It changes nothing.
func decideCopy(root string) (CopyState, *Problem) {
	if err := safepath.Check(root, projectCopy); err != nil {
		p := problemFor(projectCopy, err)
		return NoCopy, &p
	}
	dir := filepath.Join(root, filepath.FromSlash(projectCopy))
	if _, err := os.Lstat(filepath.Join(dir, "schema.yaml")); err != nil {
		return NoCopy, nil
	}
	hash, err := HashDir(dir)
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		err = &safepath.SymlinkError{Path: projectCopy + "/" + sl.Path}
		return NoCopy, &Problem{Path: projectCopy, Kind: ProblemSymlink, Err: err}
	}
	if err != nil {
		return NoCopy, &Problem{Path: projectCopy, Kind: ProblemUnreadable, Err: err}
	}
	if IsShipped(hash) {
		return ShippedCopy, nil
	}
	return EditedCopy, nil
}

// removeProjectCopy takes the project copy away: it renames the folder
// aside, so the copy is gone in one step, checks what it moved against the
// shipped hashes and the per-machine copy against the embedded schema, and
// only then deletes it. When either check fails it puts the folder back and
// removes nothing; the state says what the copy is then. It returns the
// removed files, sorted, relative to the repository root.
func removeProjectCopy(root, schemaDir string, a Assets) ([]File, CopyState, error) {
	dir := filepath.Join(root, filepath.FromSlash(projectCopy))
	aside := filepath.Join("openspec", ".docket-removed-"+strconv.Itoa(os.Getpid()))
	asideDir := filepath.Join(root, aside)
	if err := rename(dir, asideDir); err != nil {
		return nil, ShippedCopy, fmt.Errorf("remove %s: %w", projectCopy, err)
	}
	state := ShippedCopy
	copyHash, err := HashDir(asideDir)
	if err != nil || !IsShipped(copyHash) {
		state = EditedCopy
	}
	want, err := TreeHash(a.Schema)
	if err != nil {
		return nil, state, err
	}
	machine, err := HashDir(schemaDir)
	if state == EditedCopy || err != nil || machine != want {
		if err := rename(asideDir, dir); err != nil {
			return nil, state, fmt.Errorf("remove %s: the copy is renamed to %s and not put back: %w", projectCopy, aside, err)
		}
		return nil, state, nil
	}
	var files []File
	err = fs.WalkDir(os.DirFS(asideDir), ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			files = append(files, File{Path: projectCopy + "/" + p, Status: Removed, Kind: KindSchema})
		}
		return err
	})
	if err != nil {
		return nil, state, fmt.Errorf("remove %s: the copy is renamed to %s: %w", projectCopy, aside, err)
	}
	slices.SortFunc(files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	if err := removeAll(asideDir); err != nil {
		return files, state, fmt.Errorf("remove %s: the copy is renamed to %s: %w", projectCopy, aside, err)
	}
	return files, state, nil
}
