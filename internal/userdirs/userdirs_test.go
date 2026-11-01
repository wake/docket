package userdirs

import (
	"errors"
	"testing"
)

func TestDataHome(t *testing.T) {
	rows := []struct {
		name string
		env  []string
		want string
		err  error
	}{
		{"home only", []string{"HOME=/h"}, "/h/.local/share", nil},
		{"xdg wins", []string{"HOME=/h", "XDG_DATA_HOME=/d"}, "/d", nil},
		{"trailing slash kept", []string{"XDG_DATA_HOME=/d/"}, "/d/", nil},
		{"last value counts", []string{"XDG_DATA_HOME=/a", "XDG_DATA_HOME=/b"}, "/b", nil},
		{"empty xdg falls back", []string{"HOME=/h", "XDG_DATA_HOME="}, "/h/.local/share", nil},
		{"relative xdg", []string{"XDG_DATA_HOME=data"}, "", &RelativeError{"XDG_DATA_HOME", "data"}},
		{"no home", nil, "", ErrNoHome},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got, err := DataHome(r.env, nil)
			if r.err == nil && err != nil {
				t.Fatalf("error %v", err)
			}
			if r.err != nil {
				var re *RelativeError
				if want, ok := r.err.(*RelativeError); ok {
					if !errors.As(err, &re) || *re != *want {
						t.Fatalf("error %v, want %v", err, r.err)
					}
					if err.Error() != "XDG_DATA_HOME is not an absolute path: data" {
						t.Errorf("text %q", err.Error())
					}
				} else if !errors.Is(err, r.err) {
					t.Fatalf("error %v, want %v", err, r.err)
				}
			}
			if got != r.want {
				t.Errorf("got %q, want %q", got, r.want)
			}
		})
	}
}

func TestClaudeConfig(t *testing.T) {
	rows := []struct {
		name string
		env  []string
		want string
		err  error
	}{
		{"home only", []string{"HOME=/h"}, "/h/.claude", nil},
		{"config dir wins", []string{"HOME=/h", "CLAUDE_CONFIG_DIR=/c"}, "/c", nil},
		{"empty falls back", []string{"CLAUDE_CONFIG_DIR=", "HOME=/h"}, "/h/.claude", nil},
		{"relative", []string{"HOME=/h", "CLAUDE_CONFIG_DIR=c"}, "", &RelativeError{"CLAUDE_CONFIG_DIR", "c"}},
		{"config dir without home", []string{"CLAUDE_CONFIG_DIR=/c"}, "/c", nil},
		{"no home", nil, "", ErrNoHome},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got, err := ClaudeConfig(r.env, nil)
			var re *RelativeError
			switch want := r.err.(type) {
			case nil:
				if err != nil {
					t.Fatalf("error %v", err)
				}
			case *RelativeError:
				if !errors.As(err, &re) || *re != *want {
					t.Fatalf("error %v, want %v", err, r.err)
				}
			default:
				if !errors.Is(err, r.err) {
					t.Fatalf("error %v, want %v", err, r.err)
				}
			}
			if got != r.want {
				t.Errorf("got %q, want %q", got, r.want)
			}
		})
	}
}
