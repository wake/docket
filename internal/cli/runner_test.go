package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wake/docket/internal/openspec"
)

func TestOpenSpecEnvError(t *testing.T) {
	var ee EnvError
	if err := openSpecEnvError(&openspec.UnavailableError{Program: "x"}); !errors.As(err, &ee) || ee.Rule != "openspec-missing" {
		t.Errorf("unavailable: %v", err)
	}
	if err := openSpecEnvError(&openspec.UnsupportedError{}); !errors.As(err, &ee) || ee.Rule != "openspec-unsupported" {
		t.Errorf("unsupported: %v", err)
	}
	plain := errors.New("boom")
	if err := openSpecEnvError(plain); err != plain {
		t.Errorf("other: %v", err)
	}
}

func TestOpenSpecFailed(t *testing.T) {
	for _, err := range []error{&openspec.FailedError{Code: 1}, &openspec.UnsafeArgError{Program: "openspec.cmd", Arg: "a&b"}} {
		if !openSpecFailed(fmt.Errorf("wrapped: %w", err)) {
			t.Errorf("%T not a failed call", err)
		}
	}
	for _, err := range []error{&openspec.UnavailableError{Program: "x"}, errors.New("other")} {
		if openSpecFailed(err) {
			t.Errorf("%v taken as a failed call", err)
		}
	}
}
