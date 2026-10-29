// Package migrations embeds the SQL migration files so that the compiled
// server can create the database schema by itself.
package migrations

import "embed"

// FS holds every *.sql file in this folder. Files run in file-name order.
//
//go:embed *.sql
var FS embed.FS
