// Package setup installs the docket schema into a repository.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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
	if _, err := os.Stat(filepath.Join(root, "openspec")); errors.Is(err, fs.ErrNotExist) {
		if err := initOpenSpec(); err != nil {
			return res, err
		}
		res.OpenSpecInitialized = true
	} else if err != nil {
		return res, err
	}
	err := fs.WalkDir(schema, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want, err := fs.ReadFile(schema, p)
		if err != nil {
			return err
		}
		rel := path.Join(schemaDir, p)
		dst := filepath.Join(root, filepath.FromSlash(rel))
		status := Written
		got, err := os.ReadFile(dst)
		switch {
		case err == nil && bytes.Equal(got, want):
			status = Unchanged
		case err == nil && !force:
			status = Differs
		case err == nil:
			status = Overwritten
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
		if status == Written || status == Overwritten {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, want, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", rel, err)
			}
		}
		res.Files = append(res.Files, File{Path: rel, Status: status})
		return nil
	})
	return res, err
}
