package cli

// Finding is one problem a command found.
type Finding struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Change  string `json:"change,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// Unreadable names a source a command needed but could not read.
type Unreadable struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Repo identifies the repository a command ran in; nil pointers are JSON null.
type Repo struct {
	Root   string  `json:"root"`
	Head   *string `json:"head"`
	Branch *string `json:"branch"`
	Origin *string `json:"origin"`
}

// Outcome is what a command returns. Human lines are printed in text mode only.
type Outcome struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     any
	Human      []string
}

type envelope struct {
	Tool       string       `json:"tool"`
	Version    string       `json:"version"`
	Command    string       `json:"command"`
	Repo       *Repo        `json:"repo"`
	OK         bool         `json:"ok"`
	Findings   []Finding    `json:"findings"`
	Unreadable []Unreadable `json:"unreadable"`
	Result     any          `json:"result"`
}
