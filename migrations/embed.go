package migrations

import _ "embed"

// Schema holds SQL for ensuring tables exist.
//
//go:embed 001_init.sql
var Schema string
