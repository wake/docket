package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// usage prints the usage text, headed by msg when it is not empty.
func usage(w io.Writer, msg string) {
	if msg != "" {
		fmt.Fprintf(w, "dkt: %s\n", msg)
	}
	fmt.Fprintln(w, "usage: dkt <command> [args] [--json]\n       dkt --version [--json]\n       dkt --help")
}

// jsonRequested reports whether any argument is a --json flag the flag parser
// would read as true: --json, -json, or --json=<bool>.
func jsonRequested(args []string) bool {
	for _, a := range args {
		a = strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-")
		if a == "json" {
			return true
		}
		if v, ok := strings.CutPrefix(a, "json="); ok {
			if b, err := strconv.ParseBool(v); err == nil && b {
				return true
			}
		}
	}
	return false
}
