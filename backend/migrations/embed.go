// Package migrations embeds the SQL schema files into the binary so a
// deployed container migrates itself at startup with no extra tooling.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
