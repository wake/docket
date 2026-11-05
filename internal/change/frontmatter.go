package change

// Front is what ReadFrontMatter returns (stub).
type Front struct {
	Fields    map[string]Field
	Specs     map[string]string
	SpecsLine int
}

// Field is one top-level value and its 1-based line in the file.
type Field struct {
	Value string
	Line  int
}

// ReadFrontMatter is a stub.
func ReadFrontMatter(b []byte) (Front, bool, error) { return Front{}, false, nil }

// SpecTitle is a stub.
func SpecTitle(firstLine, capability string) (string, bool) { return "", false }
