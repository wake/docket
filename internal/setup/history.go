package setup

// Earlier maps a target path to the SHA-256 (lowercase hex) of the files
// earlier releases shipped there; the key for the operations section of
// openspec/config.yaml is "openspec/config.yaml#operations". Each release adds
// the hashes of what it ships. Tests in other packages may replace it, and
// must restore it.
var Earlier = map[string][]string{}
