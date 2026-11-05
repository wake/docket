package cli

import (
	"errors"

	"github.com/wake/docket/internal/openspec"
)

// openSpecRunner locates OpenSpec and checks its version; a missing or too old
// OpenSpec is an EnvError.
func openSpecRunner(c *Context) (*openspec.Runner, error) {
	r, err := openspec.New(c.Repo.Root, c.Env)
	if err == nil {
		_, err = r.CheckVersion()
	}
	var unavail *openspec.UnavailableError
	var unsup *openspec.UnsupportedError
	switch {
	case errors.As(err, &unavail):
		return nil, EnvError{Rule: "openspec-missing", Msg: err.Error()}
	case errors.As(err, &unsup):
		return nil, EnvError{Rule: "openspec-unsupported", Msg: err.Error()}
	case err != nil:
		return nil, err
	}
	return r, nil
}
