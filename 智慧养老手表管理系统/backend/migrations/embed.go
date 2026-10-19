// Package migrations contains MySQL business, PostgreSQL analytics and legacy migrations.
package migrations

import "embed"

//go:embed *.sql mysql/*.sql analytics/*.sql
var Files embed.FS
