// Package docket embeds the files dkt init writes into a project and the text it prints.
package docket

import "embed"

// Shipped holds the files `dkt init` writes into a project, and the text it prints.
//
//go:embed shipped
var Shipped embed.FS
