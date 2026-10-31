package cli

import (
	"path/filepath"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/setup"
)

// SchemaState is the docket schema dkt new and dkt archive found, as OpenSpec
// resolves it: the directory and whether it matches the shipped schema.
type SchemaState struct {
	Path    string `json:"path"`
	Matches bool   `json:"matches"`
}

// findSchema looks for the docket schema in the project, then in the
// per-machine directory. It returns a nil state when neither holds it, and an
// EnvError when the project has no copy and the per-machine directory cannot
// be located. The lines are for the text output and empty when the schema
// matches.
func findSchema(c *Context) (*SchemaState, []string, error) {
	machine, merr := setup.SchemaDir(c.Env, c.UserHome)
	if merr != nil {
		machine = ""
	}
	dir, found, err := setup.FindSchema(c.Repo.Root, machine)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		if merr != nil {
			return nil, nil, EnvError{Msg: merr.Error()}
		}
		return nil, nil, nil
	}
	hash, err := setup.HashDir(dir)
	if err != nil {
		return nil, nil, err
	}
	want, err := setup.TreeHash(docket.Schema())
	if err != nil {
		return nil, nil, err
	}
	st := &SchemaState{Path: dir, Matches: hash == want}
	if st.Matches {
		return st, nil, nil
	}
	var line string
	switch {
	case dir != filepath.Join(c.Repo.Root, "openspec/schemas/docket"):
		line = "the docket schema at " + dir + " differs from the shipped one; dkt init --force overwrites it"
	case setup.IsShipped(hash):
		line = "openspec/schemas/docket is a copy of an older shipped schema and is used instead of the per-machine one; dkt init removes it"
	default:
		line = "openspec/schemas/docket differs from every shipped schema and is used instead of the per-machine one"
	}
	return st, []string{line}, nil
}
