package main

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var dao DAO

func initDB(connString string) {
	var err error
	var driverName, dataSource string

	// Parse connection string
	if strings.HasPrefix(connString, "sqlite:///") {
		driverName = "sqlite3"
		dataSource = strings.TrimPrefix(connString, "sqlite:///")
	} else if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		driverName = "postgres"
		dataSource = connString
	} else {
		log.Fatalf("Unsupported connection string format: %s (expected sqlite:/// or postgres://)", connString)
	}

	// Open database connection
	db, err = sql.Open(driverName, dataSource)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create appropriate DAO
	if driverName == "sqlite3" {
		dao = NewSQLiteDAO(db)
	} else if driverName == "postgres" {
		dao = NewPostgresDAO(db)
	} else {
		log.Fatalf("Unsupported database driver: %s", driverName)
	}

	// Create all tables
	if err = dao.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	log.Printf("Database initialized with driver: %s", driverName)
}
