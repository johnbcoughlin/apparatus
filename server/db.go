package main

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB(connString string) {
	// Parse connection string
	// Expected format: sqlite:///path/to/db.db
	var dbPath string
	if strings.HasPrefix(connString, "sqlite:///") {
		dbPath = strings.TrimPrefix(connString, "sqlite:///")
	} else {
		log.Fatalf("Invalid connection string format. Expected sqlite:///path/to/db.db, got: %s", connString)
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	createRunsTableSQL := `
	CREATE TABLE IF NOT EXISTS runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(createRunsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create runs table: %v", err)
	}

	createParametersTableSQL := `
	CREATE TABLE IF NOT EXISTS parameters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id INTEGER NOT NULL,
		key TEXT NOT NULL,
		value_type TEXT NOT NULL,
		value_string TEXT,
		value_bool INTEGER,
		value_float REAL,
		value_int INTEGER,
		UNIQUE(run_id, key)
	);
	`

	_, err = db.Exec(createParametersTableSQL)
	if err != nil {
		log.Fatalf("Failed to create parameters table: %v", err)
	}

	createMetricsTableSQL := `
        CREATE TABLE IF NOT EXISTS metrics (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                run_id INTEGER NOT NULL,
                key TEXT NOT NULL,
                value REAL NOT NULL,
                logged_at TIMESTAMP NOT NULL,
                time REAL,
                step INTEGER,
                UNIQUE(run_id, key, time, step)
        );
        `

	_, err = db.Exec(createMetricsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create metrics table: %v", err)
	}
}
