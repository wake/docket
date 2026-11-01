package status

// OrderLine is a line of docket/order.md with the state of the idea it names.
type OrderLine struct {
	Number    int       `json:"number"`
	Path      string    `json:"path"`
	Title     *string   `json:"title"`
	State     string    `json:"state"`
	Handlers  []Handler `json:"handlers"`
	WaitingOn []string  `json:"waiting_on"`
}

// Idea is an idea the order does not name that is waiting for work or being
// worked on.
type Idea struct {
	Path     string    `json:"path"`
	Title    *string   `json:"title"`
	State    string    `json:"state"`
	Handlers []Handler `json:"handlers"`
}
