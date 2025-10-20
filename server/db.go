package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./apparatus.db")
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
}
