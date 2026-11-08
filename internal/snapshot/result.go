package snapshot

import (
	"encoding/json"
	"reflect"

	"github.com/wake/docket/internal/specdoc"
)

// The types below are the JSON of the result in design.md, Output. A pointer
// is a field that can be null; a slice is an array and encodes as [] when nil,
// except where the field is tagged null:"ok".

// Result is the whole project at one moment.
type Result struct {
	DefaultBranch *Branch      `json:"default_branch"`
	Works         []WorkEntry  `json:"works"`
	Ideas         []IdeaEntry  `json:"ideas"`
	Issues        []IssueEntry `json:"issues"`
	Order         []OrderItem  `json:"order"`
	Specs         []Spec       `json:"specs"`
	Domains       Domains      `json:"domains"`
}

// MarshalJSON encodes nil arrays as [] in the whole result; a field tagged
// null:"ok" keeps null.
func (r Result) MarshalJSON() ([]byte, error) {
	type plain Result
	p := plain(r)
	fillArrays(reflect.ValueOf(&p).Elem())
	return json.Marshal(p)
}

func fillArrays(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			fillArrays(v.Elem())
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			fillArrays(v.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Kind() == reflect.Slice && f.IsNil() && v.Type().Field(i).Tag.Get("null") != "ok" {
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			}
			fillArrays(f)
		}
	}
}

// Branch is the default branch.
type Branch struct {
	Name    string `json:"name"`
	Ref     string `json:"ref"`
	Commit  string `json:"commit"`
	Time    string `json:"time"`
	Subject string `json:"subject"`
}

// WorkEntry is one change or record.
type WorkEntry struct {
	Type       string       `json:"type"`
	Slug       string       `json:"slug"`
	Title      *string      `json:"title"`
	Kind       *string      `json:"kind"`
	Source     *string      `json:"source"`
	Schema     string       `json:"schema"`
	Created    *string      `json:"created"`
	Proposed   *string      `json:"proposed"`
	Position   *int         `json:"position"`
	ReadFrom   string       `json:"read_from"`
	Archived   *string      `json:"archived"`
	Stages     []string     `json:"stages"`
	Phase      string       `json:"phase"`
	Progress   Progress     `json:"progress"`
	Merged     bool         `json:"merged"`
	Merge      *Merge       `json:"merge"`
	Release    *Release     `json:"release"`
	LastCommit *LastCommit  `json:"last_commit"`
	Locations  []Location   `json:"locations"`
	Tasks      []Task       `json:"tasks" null:"ok"`
	Documents  []Document   `json:"documents"`
	Deltas     []DeltaEntry `json:"deltas"`
}

// Merge is the commit that brought a work into the default branch; a *Ref
// converts to *Merge (Ref has no JSON tags).
type Merge struct {
	Commit  string `json:"commit"`
	Time    string `json:"time"`
	Subject string `json:"subject"`
}

// Release is the first tag containing the merge.
type Release struct {
	Tag  string `json:"tag"`
	Time string `json:"time"`
}

// LastCommit is the newest commit naming the work.
type LastCommit struct {
	Commit string `json:"commit"`
	Time   string `json:"time"`
}

// Progress counts done tasks.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Location is one source holding a work or an issue.
type Location struct {
	Source   string  `json:"source"`
	Kind     string  `json:"kind"`
	Branch   *string `json:"branch"`
	Default  bool    `json:"default"`
	Commit   *string `json:"commit"`
	Time     *string `json:"time"`
	Path     string  `json:"path"`
	Archived bool    `json:"archived"`
}

// Task is one plan or record task.
type Task struct {
	Number  int      `json:"number"`
	Name    string   `json:"name"`
	Steps   int      `json:"steps"`
	Ticked  int      `json:"ticked"`
	Commits []string `json:"commits"`
	State   string   `json:"state"`
}

// Document is a file of a work, or its source idea or issue.
type Document struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

// DeltaEntry is one delta spec of a change.
type DeltaEntry struct {
	Capability string                `json:"capability"`
	Path       string                `json:"path"`
	Added      []specdoc.Requirement `json:"added"`
	Modified   []specdoc.Requirement `json:"modified"`
	Removed    []specdoc.Removed     `json:"removed"`
	Renamed    []specdoc.Renamed     `json:"renamed"`
}

// IdeaEntry is one idea on the default branch tip.
type IdeaEntry struct {
	Path      string    `json:"path"`
	Title     *string   `json:"title"`
	Needs     []string  `json:"needs"`
	Dropped   bool      `json:"dropped"`
	Date      *string   `json:"date"`
	State     string    `json:"state"`
	WaitingOn []string  `json:"waiting_on"`
	Position  *int      `json:"position"`
	TakenBy   []WorkRef `json:"taken_by"`
}

// IssueEntry is one issue.
type IssueEntry struct {
	Slug      string     `json:"slug"`
	Path      string     `json:"path"`
	Title     *string    `json:"title"`
	Source    *string    `json:"source"`
	Date      string     `json:"date"`
	State     string     `json:"state"`
	Position  *int       `json:"position"`
	Handlers  []Handler  `json:"handlers"`
	Locations []Location `json:"locations"`
}

// WorkRef names a work.
type WorkRef struct {
	Type string `json:"type"`
	Slug string `json:"slug"`
}

// Handler is a work taking an issue.
type Handler struct {
	Type   string `json:"type"`
	Slug   string `json:"slug"`
	Merged bool   `json:"merged"`
}

// OrderItem is one item line of docket/order.md.
type OrderItem struct {
	Position int    `json:"position"`
	Path     string `json:"path"`
}

// Spec is one main spec.
type Spec struct {
	Capability   string   `json:"capability"`
	Title        string   `json:"title"`
	Path         string   `json:"path"`
	Requirements []string `json:"requirements"`
}

// Domains is the domain tree.
type Domains struct {
	Found        bool     `json:"found"`
	Nodes        []Node   `json:"nodes"`
	Unclassified []string `json:"unclassified"`
}

// Node is one node of the domain tree.
type Node struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Children     []Node   `json:"children"`
}
