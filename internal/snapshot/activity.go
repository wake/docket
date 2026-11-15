package snapshot

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/wake/docket/internal/activity"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/projects"
	"github.com/wake/docket/internal/scan"
)

// resolve follows links in p; a path that cannot be resolved stays as it is.
func resolve(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// setActivity gives each unmerged, unarchived work in a worktree the strongest state among the
// live records whose directory is deepest in that worktree. It only reads the sessions folder.
func setActivity(env []string, wts []gitx.Worktree, works []WorkEntry) []scan.Unreadable {
	recs, err := activity.Read(env, time.Now())
	if err != nil {
		src, _ := projects.SessionsDir(env)
		return []scan.Unreadable{{Source: src, Reason: err.Error()}}
	}
	type cand struct{ path, resolved string }
	var cands []cand
	for _, w := range wts {
		if !w.Bare && !w.Prunable {
			cands = append(cands, cand{w.Path, resolve(w.Path)})
		}
	}
	state := map[string]string{} // worktree path as listed -> "working" or "waiting"
	for _, r := range recs {
		best, dir := -1, resolve(r.Dir)
		for i, c := range cands {
			if (dir == c.resolved || strings.HasPrefix(dir, c.resolved+string(filepath.Separator))) &&
				(best < 0 || len(c.resolved) > len(cands[best].resolved)) {
				best = i
			}
		}
		if best >= 0 && state[cands[best].path] != "working" {
			state[cands[best].path] = r.State
		}
	}
	for i := range works {
		w := &works[i]
		if w.Merged {
			continue
		}
		for _, l := range w.Locations {
			if s, ok := state[l.Source]; ok && l.Kind == "worktree" && !l.Archived && (w.Activity == nil || s == "working") {
				w.Activity = &s
			}
		}
	}
	return nil
}
