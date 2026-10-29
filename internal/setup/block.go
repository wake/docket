package setup

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	blockStart = "<!-- docket:start -->"
	blockEnd   = "<!-- docket:end -->"
)

// errMalformed is wrapped by every error FindBlock returns for markers that
// do not form exactly one block.
var errMalformed = errors.New("malformed docket block")

func malformed(why string) error {
	return fmt.Errorf("%w: %s", errMalformed, why)
}

// FindBlock locates the Docket block in b: from the start of the start marker
// line through the end of the end marker text, without the newline after it.
// A marker line is exactly the marker, with at most one CR before the LF.
// found is false with a nil error when b has no marker; any shape other than
// one start marker followed by one end marker is errMalformed.
func FindBlock(b []byte) (from, to int, found bool, err error) {
	start, end := -1, -1
	for off := 0; off < len(b); {
		nl := bytes.IndexByte(b[off:], '\n')
		lineEnd, next := len(b), len(b)
		if nl >= 0 {
			lineEnd, next = off+nl, off+nl+1
		}
		line := bytes.TrimSuffix(b[off:lineEnd], []byte("\r"))
		switch string(line) {
		case blockStart:
			if start >= 0 {
				return 0, 0, false, malformed("more than one block")
			}
			start = off
		case blockEnd:
			if start < 0 {
				return 0, 0, false, malformed("end marker before start marker")
			}
			if end >= 0 {
				return 0, 0, false, malformed("more than one block")
			}
			end = off + len(line)
		}
		off = next
	}
	switch {
	case start < 0:
		return 0, 0, false, nil
	case end < 0:
		return 0, 0, false, malformed("no end marker")
	}
	return start, end, true, nil
}

// placeBlock returns the bytes to write for a file holding cur (nil when the
// file is missing) so that it carries block. next is nil when
// nothing is to be written. A block that differs is left alone (Differs)
// unless force is set (Overwritten).
func placeBlock(cur []byte, block string, force bool) (next []byte, status FileStatus, err error) {
	from, to, found, err := FindBlock(cur)
	if err != nil {
		return nil, "", err
	}
	if found {
		if string(cur[from:to]) == block {
			return nil, Unchanged, nil
		}
		if !force {
			return nil, Differs, nil
		}
		next = append(next, cur[:from]...)
		next = append(next, block...)
		next = append(next, cur[to:]...)
		return next, Overwritten, nil
	}
	next = append(next, cur...)
	if len(cur) > 0 {
		if cur[len(cur)-1] != '\n' {
			next = append(next, '\n')
		}
		next = append(next, '\n')
	}
	next = append(next, block...)
	next = append(next, '\n')
	return next, Written, nil
}

// importsAgents reports whether a line of b, with surrounding spaces and tabs
// removed, is @AGENTS.md or @./AGENTS.md.
func importsAgents(b []byte) bool {
	for _, l := range bytes.Split(b, []byte("\n")) {
		switch string(bytes.Trim(l, " \t\r")) {
		case "@AGENTS.md", "@./AGENTS.md":
			return true
		}
	}
	return false
}
