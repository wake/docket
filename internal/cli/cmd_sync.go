package cli

type syncResult struct {
	Slug  string     `json:"slug"`
	Check bool       `json:"check"`
	Tasks []syncTask `json:"tasks"` // never null
}

type syncTask struct {
	Number  int      `json:"number"`
	Name    string   `json:"name"`
	Commits []string `json:"commits"` // full SHAs, oldest first; never null
	Steps   int      `json:"steps"`
	Done    int      `json:"done"`   // ticked steps after the run (as the file is, with --check)
	Ticked  int      `json:"ticked"` // steps this run ticked (or would, with --check)
}

func runSync(c *Context, slug string, check bool) (Outcome, error) {
	return Outcome{}, nil
}
