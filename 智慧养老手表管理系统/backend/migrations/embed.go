// Package migrations embeds the active MySQL business and PostgreSQL analytics schemas.
package migrations

import "embed"

//go:embed mysql/*.sql analytics/*.sql
var Files embed.FS
