package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func main() {
	initDB()

	// Define routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/runs", handleAPICreateRun)
	http.HandleFunc("/runs/", handleViewRun)

	// Start server
	port := "8080"
	log.Printf("Starting Apparatus server on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Apparatus</title>
</head>
<body>
	<h1>Welcome to Apparatus</h1>
	<p>Experiment tracking without the AI cruft.</p>
</body>
</html>`)
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

func handleViewRun(w http.ResponseWriter, r *http.Request) {
	runUUID := strings.TrimPrefix(r.URL.Path, "/runs/")

	var name string
	err := db.QueryRow("SELECT name FROM runs WHERE uuid = ?", runUUID).Scan(&name)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Run: %s</title>
</head>
<body>
	<h1>Run: %s</h1>
	<p>UUID: %s</p>
</body>
</html>`, name, name, runUUID)
}
