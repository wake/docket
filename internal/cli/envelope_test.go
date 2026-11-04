package cli

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCode(t *testing.T) {
	f := []Finding{{Rule: "r"}}
	u := []Unreadable{{Source: "s"}}
	cases := []struct {
		name string
		o    Outcome
		err  error
		want int
	}{
		{"clean", Outcome{}, nil, 0},
		{"findings", Outcome{Findings: f}, nil, 1},
		{"unreadable", Outcome{Unreadable: u}, nil, 3},
		{"both", Outcome{Findings: f, Unreadable: u}, nil, 3},
		{"usage", Outcome{}, UsageError{Msg: "x"}, 2},
		{"wrapped usage", Outcome{}, fmt.Errorf("w: %w", UsageError{Msg: "x"}), 2},
		{"env", Outcome{}, EnvError{Msg: "x"}, 4},
		{"other", Outcome{}, errors.New("boom"), 1},
	}
	for _, c := range cases {
		if got := ExitCode(c.o, c.err); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
