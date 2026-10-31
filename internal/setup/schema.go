package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/userdirs"
)

// ShippedSchemas are the tree hashes of every docket schema a release has
// shipped, oldest first.
var ShippedSchemas = []string{
	"48a66028d9fad9ca08920ddd8839a475ae1f1405af473ddb7d190a8562f455c3", // v0.1.0 to v0.3.0
	"e4636dc03a3cf7e32be049fd019d9b7a51c44dbe9ea3cc67a21ffbe1c1b7cec1", // v0.4.0 to v0.4.2
	"edc92ac411c0e925656d58d8ac7de51b08bdd47b9e92356c4cca073b58bcefde", // v0.5.0 to v0.9.1
	"886ab9e7c68599e1049119c8ae6616629a513b1319602f15ecb2643b8e951c21", // v0.9.2
}

// IsShipped reports whether hash is the tree hash of a schema Docket shipped.
func IsShipped(hash string) bool { return slices.Contains(ShippedSchemas, hash) }

// SchemaDir returns the per-machine directory OpenSpec reads the docket
// schema from: <data home>/openspec/schemas/docket.
func SchemaDir(env []string, userHome func() (string, error)) (string, error) {
	data, err := userdirs.DataHome(env, userHome)
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "openspec", "schemas", "docket"), nil
}

// TreeHash returns the lowercase hex SHA-256 over the regular files of fsys,
// sorted by slash path: each contributes its path, a NUL, its length in
// decimal, a NUL and its bytes.
func TreeHash(fsys fs.FS) (string, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		files[p] = b
		return err
	})
	if err != nil {
		return "", err
	}
	return hashFiles(files), nil
}

// HashDir returns the TreeHash of the files under dir, read without
// following links. A link anywhere under dir is a *safepath.SymlinkError
// whose Path is relative to dir.
func HashDir(dir string) (string, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.Type()&fs.ModeSymlink != 0 {
			return &safepath.SymlinkError{Path: rel}
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := readNoFollow(p)
		files[rel] = b
		return err
	})
	if err != nil {
		return "", err
	}
	return hashFiles(files), nil
}

func hashFiles(files map[string][]byte) string {
	h := sha256.New()
	for _, p := range slices.Sorted(maps.Keys(files)) {
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write([]byte(strconv.Itoa(len(files[p]))))
		h.Write([]byte{0})
		h.Write(files[p])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FindSchema returns the directory dkt reads the docket schema from: the
// project's openspec/schemas/docket when it holds schema.yaml, else
// machineDir when it does (an empty machineDir is skipped).
func FindSchema(root, machineDir string) (dir string, found bool, err error) {
	for _, d := range []string{filepath.Join(root, "openspec", "schemas", "docket"), machineDir} {
		if d == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(d, "schema.yaml"))
		switch {
		case err == nil && info.Mode().IsRegular():
			return d, true, nil
		case err == nil || errors.Is(err, fs.ErrNotExist):
		default:
			return "", false, err
		}
	}
	return "", false, nil
}
