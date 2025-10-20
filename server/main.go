package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates *template.Template

func main() {
	initDB()

	// Load templates from embedded filesystem
	var err error
	templates, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Define routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/runs", handleAPICreateRun)
	http.HandleFunc("/api/params", handleAPILogParam)
	http.HandleFunc("/runs/", handleViewRun)

	// Start server
	port := "8080"
	log.Printf("Starting Apparatus server on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

type Run struct {
	UUID      string
	Name      string
	CreatedAt string
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Query all runs
	rows, err := db.Query(`
		SELECT uuid, name, created_at
		FROM runs
		ORDER BY created_at DESC`)
	if err != nil {
		log.Fatalf("Failed to query runs: %v", err)
	}
	defer rows.Close()

	var runs []Run

	for rows.Next() {
		var uuid, name, createdAt string
		err := rows.Scan(&uuid, &name, &createdAt)
		if err != nil {
			log.Fatalf("Failed to scan run: %v", err)
		}
		runs = append(runs, Run{UUID: uuid, Name: name, CreatedAt: createdAt})
	}

	data := struct {
		Title string
		Runs  []Run
	}{
		Title: "Home",
		Runs:  runs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func handleAPICreateRun(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	runUUID := uuid.New().String()

	_, err := db.Exec("INSERT INTO runs (uuid, name) VALUES (?, ?)", runUUID, name)
	if err != nil {
		log.Fatalf("Failed to insert run: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   runUUID,
		"name": name,
	})
}

func handleAPILogParam(w http.ResponseWriter, r *http.Request) {
	runUUID := r.URL.Query().Get("run_uuid")
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	valueType := r.URL.Query().Get("type")

	// Get run_id from uuid
	var runID int
	err := db.QueryRow("SELECT id FROM runs WHERE uuid = ?", runUUID).Scan(&runID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Insert parameter based on type
	var insertSQL string
	var args []interface{}

	switch valueType {
	case "string":
		insertSQL = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_string) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, value}
	case "bool":
		boolValue := 0
		if value == "true" {
			boolValue = 1
		}
		insertSQL = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_bool) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, boolValue}
	case "float":
		insertSQL = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_float) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, value}
	case "int":
		insertSQL = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_int) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, value}
	}

	_, err = db.Exec(insertSQL, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type Parameter struct {
	Key   string
	Value string
	Type  string
}

func handleViewRun(w http.ResponseWriter, r *http.Request) {
	runUUID := strings.TrimPrefix(r.URL.Path, "/runs/")

	var name string
	var runID int
	err := db.QueryRow("SELECT id, name FROM runs WHERE uuid = ?", runUUID).Scan(&runID, &name)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}

	// Query parameters for this run
	rows, err := db.Query(`
		SELECT key, value_type, value_string, value_bool, value_float, value_int
		FROM parameters
		WHERE run_id = ?
		ORDER BY key`, runID)
	if err != nil {
		log.Fatalf("Failed to query parameters: %v", err)
	}
	defer rows.Close()

	var parameters []Parameter

	for rows.Next() {
		var key, valueType string
		var valueString sql.NullString
		var valueBool sql.NullInt64
		var valueFloat sql.NullFloat64
		var valueInt sql.NullInt64

		err := rows.Scan(&key, &valueType, &valueString, &valueBool, &valueFloat, &valueInt)
		if err != nil {
			log.Fatalf("Failed to scan parameter: %v", err)
		}

		var value string
		switch valueType {
		case "string":
			value = valueString.String
		case "bool":
			if valueBool.Int64 == 1 {
				value = "true"
			} else {
				value = "false"
			}
		case "float":
			value = fmt.Sprintf("%g", valueFloat.Float64)
		case "int":
			value = fmt.Sprintf("%d", valueInt.Int64)
		}

		parameters = append(parameters, Parameter{Key: key, Value: value, Type: valueType})
	}

	data := struct {
		Title      string
		UUID       string
		Name       string
		Parameters []Parameter
	}{
		Title:      name,
		UUID:       runUUID,
		Name:       name,
		Parameters: parameters,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.ExecuteTemplate(w, "run.html", data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}
