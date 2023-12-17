package database

import (
	"database/sql"
	"fmt"
	"time"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

func PingDB(db *sql.DB) error {
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}
	return nil
}
