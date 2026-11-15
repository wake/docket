// Package activity keeps one small record per coding agent session under the state home,
// so that dkt snapshot can tell which works a session is on.
package activity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/wake/docket/internal/projects"
	"github.com/wake/docket/internal/safefile"
)

// Window is how long a record counts after it was written.
const Window = 15 * time.Minute

// Record is one session's last reported state.
type Record struct {
	Session string    `json:"session"`
	Dir     string    `json:"dir"`
	State   string    `json:"state"`
	Time    time.Time `json:"time"`
}

// Write records state for the session named in the hook input in (JSON, may be nil), or for
// the directory wd when the input names none; "ended" removes the record. Files older than
// Window are removed too. The caller may ignore the error: it only tells why nothing was written.
func Write(env []string, now time.Time, state string, in io.Reader, wd string) error {
	if state != "working" && state != "waiting" && state != "ended" {
		return fmt.Errorf("unknown state %q", state)
	}
	dir, err := projects.SessionsDir(env)
	if err != nil {
		return err
	}
	var hook struct {
		SessionID string `json:"session_id"`
		Cwd       string `json:"cwd"`
	}
	if in != nil {
		_ = json.NewDecoder(in).Decode(&hook) // input that is not JSON is as good as none
	}
	d := hook.Cwd
	if d == "" {
		d = wd
	}
	if abs, err := filepath.Abs(d); err == nil {
		d = abs
	}
	if r, err := filepath.EvalSymlinks(d); err == nil {
		d = r
	}
	seed := "dir:" + d
	if hook.SessionID != "" {
		seed = "session:" + hook.SessionID
	}
	sum := sha256.Sum256([]byte(seed))
	path := filepath.Join(dir, hex.EncodeToString(sum[:])+".json")

	var first error
	if state == "ended" {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			first = err
		}
	} else {
		data, _ := json.Marshal(struct {
			Session string `json:"session"`
			Dir     string `json:"dir"`
			State   string `json:"state"`
			Time    string `json:"time"`
		}{hook.SessionID, d, state, now.UTC().Format(time.RFC3339)})
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		first = safefile.Install(path, append(data, '\n'), 0o600, false, path)
	}
	if es, err := os.ReadDir(dir); err == nil {
		for _, e := range es {
			cutoff := now.Add(-Window)
			if fi, err := e.Info(); err == nil && fi.Mode().IsRegular() && fi.ModTime().Before(cutoff) {
				if err := removeIfOld(filepath.Join(dir, e.Name()), cutoff); err != nil && first == nil {
					first = err
				}
			}
		}
	}
	return first
}

// Read returns the records written within Window before now. A missing folder or no home gives
// none; a folder that cannot be listed is an error. It never writes.
func Read(env []string, now time.Time) ([]Record, error) {
	dir, err := projects.SessionsDir(env)
	if err != nil {
		return nil, nil
	}
	es, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var recs []Record
	for _, e := range es {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := safefile.ReadNoFollow(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var r Record
		if json.Unmarshal(data, &r) != nil || (r.State != "working" && r.State != "waiting") || r.Time.IsZero() {
			continue
		}
		if d := now.Sub(r.Time); d <= Window && d >= -Window {
			recs = append(recs, r)
		}
	}
	return recs, nil
}

// removeIfOld removes path when it is older than cutoff, without the check-then-remove race: a
// record renamed onto path after the caller's check is moved aside, found newer and linked back.
func removeIfOld(path string, cutoff time.Time) error {
	aside := fmt.Sprintf("%s.prune-%d-%d", path, os.Getpid(), time.Now().UnixNano())
	if err := os.Rename(path, aside); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer os.Remove(aside)
	fi, err := os.Lstat(aside)
	if err != nil {
		return err
	}
	if !fi.ModTime().Before(cutoff) {
		// Fails when a newer record already took path, which is fine.
		os.Link(aside, path)
	}
	return nil
}
