// Package migrations contains the versioned PostgreSQL schema shipped with the server.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
