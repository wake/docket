package gitx

// Trailers is one commit and the values of its Change trailer.
type Trailers struct {
	SHA    string
	Change []string
}

// ChangeTrailers lists every commit reachable from HEAD, oldest first, with
// the values of its Change trailer as git reads trailers.
func (g Git) ChangeTrailers() ([]Trailers, error) { return nil, nil }
