package conflicts

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
)

// Proposal is the proposal.md of every scratch copy. OpenSpec refuses a change
// whose proposal has no Why section, which the proposal of a new change lacks.
const Proposal = `# Proposal

This scratch change is written by dkt conflicts so that OpenSpec can parse a copy of a change's delta specs.

## Why

Parse the delta specs of a change.

## What Changes

- Nothing: this copy is deleted after it is parsed.
`

// Touch is one requirement that a change adds, modifies, removes or renames.
// A rename gives two touches, RENAMED-FROM and RENAMED-TO.
type Touch struct{ Capability, Requirement, Operation string }

// Parsed is a Copy with the requirements it touches.
type Parsed struct {
	Copy    Copy
	Touches []Touch
}

// Show parses the change slug in the OpenSpec project at dir.
type Show func(dir, slug string) ([]openspec.Delta, error)

const workers = 4

// Parse writes each copy into a scratch OpenSpec project and parses it with
// show. A copy that cannot be parsed is reported once per source and gives no
// Parsed; the error is for failures that are not about one copy.
func Parse(copies []Copy, show Show) ([]Parsed, []scan.Unreadable, error) {
	type result struct {
		touches []Touch
		err     error
	}
	results := make([]result, len(copies))
	var fatal error
	var once sync.Once
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, len(copies)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				var f error
				results[i].touches, results[i].err, f = parseOne(copies[i], show)
				if f != nil {
					once.Do(func() { fatal = f })
				}
			}
		}()
	}
	for i := range copies {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	if fatal != nil {
		return nil, nil, fatal
	}

	var parsed []Parsed
	var unreadable []scan.Unreadable
	for i, r := range results {
		if r.err != nil {
			for _, s := range copies[i].Sources {
				unreadable = append(unreadable, scan.Unreadable{Source: s + ":" + copies[i].Folder, Reason: r.err.Error()})
			}
			continue
		}
		parsed = append(parsed, Parsed{Copy: copies[i], Touches: r.touches})
	}
	return parsed, unreadable, nil
}

// parseOne returns the error of the copy, or fatal when no copy could be parsed.
func parseOne(c Copy, show Show) (touches []Touch, copyErr, fatal error) {
	dir, err := os.MkdirTemp("", "dkt-conflicts-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)
	root := filepath.Join(dir, "openspec", "changes", c.Slug)
	files := map[string][]byte{"proposal.md": []byte(Proposal)}
	for p, b := range c.Files {
		files[filepath.Join("specs", filepath.FromSlash(p))] = b
	}
	for p, b := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err, nil
		}
		if err := os.WriteFile(full, b, 0o644); err != nil {
			return nil, err, nil
		}
	}
	deltas, err := show(dir, c.Slug)
	if err != nil {
		return nil, err, nil
	}
	for _, d := range deltas {
		if d.Operation == "RENAMED" && d.Rename != nil {
			touches = append(touches,
				Touch{d.Spec, d.Rename.From, "RENAMED-FROM"},
				Touch{d.Spec, d.Rename.To, "RENAMED-TO"})
			continue
		}
		for _, r := range d.Requirements {
			touches = append(touches, Touch{d.Spec, r.Name, d.Operation})
		}
	}
	return touches, nil, nil
}
