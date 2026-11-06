package cli

import (
	"errors"
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
