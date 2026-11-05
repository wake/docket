package setup

// earlier maps a target path to the SHA-256 (lowercase hex) of the files
// earlier releases shipped there. Each release adds the hashes of what it ships.
var earlier = map[string][]string{}
