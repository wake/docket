package setup

import "testing"

func TestSchemaKey(t *testing.T) {
	ok := []struct {
		in   string
		val  string
		line int
	}{
		{"schema: spec-driven\n# x\n", "spec-driven", 0},
		{"context: |\n  schema: no\nschema: docket\n", "docket", 2},
		{"# c\n", "", -1},
		{"schema: \"spec-driven\" # why\n", "spec-driven", 0},
		{"schema: 'docket'\n", "docket", 0},
		{"schema: docket\r\n", "docket", 0},
	}
	for _, c := range ok {
		v, l, err := SchemaKey([]byte(c.in))
		if err != nil || v != c.val || l != c.line {
			t.Errorf("%q = %q, %d, %v; want %q, %d", c.in, v, l, err, c.val, c.line)
		}
	}
	for _, in := range []string{"schema:\n  docket\n", "schema: a\nschema: b\n", "schema: a b\n",
		"\"schema\": spec-driven\n", "'schema': x\n", "schema : spec-driven\n", "\ufeffschema: spec-driven\n", "{schema: x}\n", "schema: [\n"} {
		if _, _, err := SchemaKey([]byte(in)); err == nil {
			t.Errorf("%q: no error", in)
		}
	}
}

func TestNameDocket(t *testing.T) {
	for _, c := range [][2]string{
		{"schema: spec-driven\n# x\n", "schema: docket\n# x\n"},
		{"schema: spec-driven # keep\r\nrules: {}\r\n", "schema: docket # keep\r\nrules: {}\r\n"},
		{"schema: \"spec-driven\"\n", "schema: docket\n"},
		{"context: x\n", "schema: docket\ncontext: x\n"},
		{"", "schema: docket\n"},
	} {
		got, err := NameDocket([]byte(c[0]))
		if err != nil || string(got) != c[1] {
			t.Errorf("%q = %q, %v; want %q", c[0], got, err, c[1])
		}
	}
}
