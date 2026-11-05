package openspec_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/wake/docket/internal/openspec"
)

func TestShowSpecWithoutRequirements(t *testing.T) {
	r := jsonRunner(t, 0, `{}`, specArgs)
	if _, err := r.ShowSpec("booking"); err == nil || !strings.Contains(err.Error(), "openspec show booking") {
		t.Fatalf("err = %v", err)
	}
	r = jsonRunner(t, 0, `{"requirements":[]}`, specArgs)
	if got, err := r.ShowSpec("booking"); err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestValidateInvalidWithoutError(t *testing.T) {
	r := jsonRunner(t, 1, `{"items":[{"id":"add-x","valid":false}]}`, validArgs)
	_, err := r.Validate("add-x")
	var fe *openspec.FailedError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v", err)
	}
	r = jsonRunner(t, 0, `{"items":[{"id":"add-x","valid":false}]}`, validArgs)
	if _, err := r.Validate("add-x"); err == nil || !strings.Contains(err.Error(), "openspec validate add-x") {
		t.Fatalf("err = %v", err)
	}
	r = jsonRunner(t, 1, `{"items":[{"id":"add-x","valid":false,"issues":[{"level":"WARNING","path":"file","message":"w"}]}]}`, validArgs)
	if got, err := r.Validate("add-x"); err == nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
	r = jsonRunner(t, 0, `{"items":[{"id":"add-x","valid":true,"issues":[{"level":"WARNING","path":"file","message":"w"}]}]}`, validArgs)
	if got, err := r.Validate("add-x"); err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
}
