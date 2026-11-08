package projects

import (
	"errors"
	"fmt"
	"time"

	"github.com/wake/docket/internal/gitx"
)

// Register adds the Docket project at root to the index unless the settings turn that off,
// the project or its main worktree is already listed, or root is not a Docket project.
func Register(root string, g gitx.Git, env []string, now time.Time) error {
	if !IsDocket(root) {
		return nil
	}
	noHome := fmt.Errorf("cannot locate the user folders: set HOME, XDG_CONFIG_HOME and XDG_STATE_HOME")
	settings, err := SettingsPath(env)
	if errors.Is(err, ErrNoHome) {
		return noHome
	}
	if err != nil {
		return err
	}
	on, err := AutoRegister(settings)
	if err != nil || !on {
		return err
	}
	path, err := IndexPath(env)
	if errors.Is(err, ErrNoHome) {
		return noHome
	}
	if err != nil {
		return err
	}
	x, err := Load(path)
	if err != nil {
		return err
	}
	if x.Find(root) >= 0 {
		return nil
	}
	main, err := ProjectPath(g, root)
	if err != nil {
		return err
	}
	if x.Find(main) >= 0 {
		return nil
	}
	origin, err := Origin(g)
	if err != nil {
		return err
	}
	e := Entry{Path: main, Added: Stamp(now), Origin: origin}
	x.Projects = append(x.Projects, e)
	return x.Save(path)
}
