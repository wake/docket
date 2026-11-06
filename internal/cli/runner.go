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
	if err != nil {
		return nil, openSpecEnvError(err)
	}
	return r, nil
}

// openSpecEnvError turns a missing or too old OpenSpec into an EnvError and returns any other err as is.
func openSpecEnvError(err error) error {
	var unavail *openspec.UnavailableError
	var unsup *openspec.UnsupportedError
	switch {
	case errors.As(err, &unavail):
		return EnvError{Rule: "openspec-missing", Msg: err.Error()}
	case errors.As(err, &unsup):
		return EnvError{Rule: "openspec-unsupported", Msg: err.Error()}
	}
	return err
}
