// core/db/embed.go
package dbschema

import _ "embed" // must be a blank import for go:embed with string

//go:embed schema.sql
var Schema string
