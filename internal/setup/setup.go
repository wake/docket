// Package setup installs the docket schema into a repository.
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

// FileStatus says what init did with one schema file.
type FileStatus string

const (
	Written     FileStatus = "written"
	Unchanged   FileStatus = "unchanged"
	Differs     FileStatus = "differs"
	Overwritten FileStatus = "overwritten"
)

// File is one schema file; Path is relative to the repository root.
type File struct {
	Path   string     `json:"path"`
	Status FileStatus `json:"status"`
}

// Result is what Init reports.
type Result struct {
	OpenSpecInitialized bool   `json:"openspec_initialized"`
	Files               []File `json:"files"`
	AgentsSection       string `json:"agents_section"`
}

// schemaDir is where the schema lives, relative to the repository root.
const schemaDir = "openspec/schemas/docket"

// Init writes the files of schema under <root>/openspec/schemas/docket. When
// <root>/openspec does not exist it first calls initOpenSpec. A differing file
// is left alone (Differs) unless force is set (Overwritten).
func Init(root string, schema fs.FS, force bool, initOpenSpec func() error) (Result, error) {
	res := Result{Files: []File{}}
	if err := safepath.Check(root, "openspec"); err != nil {
		return res, err
	}
	if _, err := os.Lstat(filepath.Join(root, "openspec")); errors.Is(err, fs.ErrNotExist) {
		if err := initOpenSpec(); err != nil {
			return res, err
		}
		res.OpenSpecInitialized = true
	} else if err != nil {
		return res, err
	}
	var names []string
	err := fs.WalkDir(schema, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, p)
		}
		return err
	})
	if err != nil {
		return res, err
	}
	// Refuse links before writing anything.
	for _, p := range names {
		if err := safepath.Check(root, path.Join(schemaDir, p)); err != nil {
			return res, err
		}
	}
	for _, p := range names {
		want, err := fs.ReadFile(schema, p)
		if err != nil {
			return res, err
		}
		rel := path.Join(schemaDir, p)
		status, err := install(filepath.Join(root, filepath.FromSlash(rel)), rel, want, force)
		if err != nil {
			return res, err
		}
		res.Files = append(res.Files, File{Path: rel, Status: status})
	}
	return res, nil
}

// install writes want to dst unless it is there already, never following a
// link at dst.
func install(dst, rel string, want []byte, force bool) (FileStatus, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	status := Overwritten
	f, err := os.OpenFile(dst, os.O_RDWR|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, fs.ErrNotExist) {
		status = Written
		f, err = os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
	}
	if err != nil {
		return "", fmt.Errorf("open %s: %w", rel, err)
	}
	defer f.Close()
	if status == Overwritten {
		got, err := io.ReadAll(f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", rel, err)
		}
		if bytes.Equal(got, want) {
			return Unchanged, nil
		}
		if !force {
			return Differs, nil
		}
		if err := f.Truncate(0); err != nil {
			return "", fmt.Errorf("write %s: %w", rel, err)
		}
	}
	if _, err := f.WriteAt(want, 0); err != nil {
		return "", fmt.Errorf("write %s: %w", rel, err)
	}
	return status, nil
}
