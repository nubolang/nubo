package sql

import (
	"database/sql"
	"time"
)

func configureDBConnection(db *sql.DB, driver string) {
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Driver-specific initial queries can be done here if needed
	if driver == "sqlite3" {
		db.Exec("PRAGMA foreign_keys = ON")
		db.Exec("PRAGMA busy_timeout = 5000")
	}
}
