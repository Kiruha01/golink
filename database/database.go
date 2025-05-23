package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Database struct {
	*sql.DB
}

// NewDatabase initializes a new database connection
func NewDatabase(host string, port int, user, password, dbname string) (*Database, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			original_url TEXT NOT NULL,
			short_code TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Database{db}, nil
}
