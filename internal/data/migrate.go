package data

import (
	"database/sql"
	"embed"
)

//go:embed migrations/001_init.sql
var migrationsFS embed.FS

func ensureSchema(db *sql.DB) error {
	ddlBytes, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(ddlBytes))
	return err
}
