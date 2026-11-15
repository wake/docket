package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/wake/docket/internal/activity"
)

const activityUsage = "activity <working|waiting|ended>"

// activityCommand exists so that usage lists activity; RunWith hands the call to runActivity
// before any command runs.
func activityCommand() Command { return Command{Name: "activity", Usage: activityUsage} }

// runActivity is the hook call: it records a session's state, prints nothing, starts no
// process, registers no project and returns 0 whatever happens. Only --help prints.
func runActivity(o Options, args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(o.Stdout, "usage: dkt %s\n", activityUsage)
		return 0
	}
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		return 0
	}
	now := time.Now()
	if o.Now != nil {
		now = o.Now()
	}
	_ = activity.Write(o.Env, now, args[0], o.Stdin, o.Dir)
	return 0
}

// IsActivity reports whether args call dkt activity: the first argument that is not a flag.
func IsActivity(args []string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a == "activity"
		}
	}
	return false
}
