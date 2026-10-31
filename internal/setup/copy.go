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
// aside, so the copy is gone in one step, then deletes it. It returns the
// removed files, sorted, relative to the repository root.
func removeProjectCopy(root string) ([]File, error) {
	dir := filepath.Join(root, filepath.FromSlash(projectCopy))
	var files []File
	err := fs.WalkDir(os.DirFS(dir), ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			files = append(files, File{Path: projectCopy + "/" + p, Status: Removed, Kind: KindSchema})
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("remove %s: %w", projectCopy, err)
	}
	slices.SortFunc(files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	aside := filepath.Join("openspec", ".docket-removed-"+strconv.Itoa(os.Getpid()))
	if err := rename(dir, filepath.Join(root, aside)); err != nil {
		return nil, fmt.Errorf("remove %s: %w", projectCopy, err)
	}
	if err := removeAll(filepath.Join(root, aside)); err != nil {
		return files, fmt.Errorf("remove %s: the copy is renamed to %s: %w", projectCopy, aside, err)
	}
	return files, nil
}
