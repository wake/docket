package change

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"go.yaml.in/yaml/v3"
)

// The two metadata files of a change folder.
const (
	DocketFile   = "docket.yaml"
	OpenSpecFile = ".openspec.yaml"
)

// Docket is the content of a change's docket.yaml.
type Docket struct {
	Title  string `yaml:"title"`
	Kind   string `yaml:"kind,omitempty"`
	Source string `yaml:"source,omitempty"`
	Base   string `yaml:"base,omitempty"`
}

// ValidTitle reports whether s can be a title: not blank, on one line.
func ValidTitle(s string) bool {
	return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\r\n")
}

// ParseDocket reads docket.yaml. It fails when b is not a YAML mapping or a
// key does not decode as a string. An invalid title reads as "".
func ParseDocket(b []byte) (Docket, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Docket{}, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return Docket{}, fmt.Errorf("not a YAML mapping")
	}
	var d Docket
	if err := doc.Content[0].Decode(&d); err != nil {
		return Docket{}, err
	}
	if !ValidTitle(d.Title) {
		d.Title = ""
	}
	return d, nil
}

// Marshal writes the keys in struct order and leaves the empty ones out.
func (d Docket) Marshal() ([]byte, error) {
	return yaml.Marshal(d)
}

// CreateDocket creates path, which must not exist and is not followed when it
// is a symbolic link, and writes d to it. It fails when the file does not
// read back as d; the file is then left for the caller to remove.
func CreateDocket(path string, d Docket) error {
	b, err := d.Marshal()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(b)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	back, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	got, err := ParseDocket(back)
	if err != nil {
		return err
	}
	if got != d {
		return fmt.Errorf("%s did not read back as written", DocketFile)
	}
	return nil
}
