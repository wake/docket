module github.com/wake/docket

go 1.25

// Windows reports junctions as symbolic links, so the checks for links catch them.
godebug winsymlink=0
