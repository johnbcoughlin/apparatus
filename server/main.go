package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
        "time"

	"github.com/google/uuid"
)

func main() {
	// Parse command line flags
	dbConnString := flag.String("db", "sqlite:///apparatus.db", "Database connection string (e.g., sqlite:///path/to/db.db)")
        artifactStoreURI := flag.String("artifact-store-uri", "file://artifacts", "URI for location to store artifacts (e.g. file:///path/to/artifacts")
	flag.Parse()

	initDB(*dbConnString)
	initArtifactStore(*artifactStoreURI)

	// Define routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/runs", handleAPICreateRun)
	http.HandleFunc("/api/params", handleAPILogParam)
	http.HandleFunc("/api/metrics", handleAPILogMetric)
        http.HandleFunc("/api/artifacts", handleAPILogArtifact)
	http.HandleFunc("/runs/", handleViewRun)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

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
	tmpl, err := template.ParseFiles("templates/header.html", "templates/home.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.ExecuteTemplate(w, "home.html", data)
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

func handleAPILogMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RunUUID  string   `json:"run_uuid"`
		Key      string   `json:"key"`
		Value    *float64 `json:"value,omitempty"`
		LoggedAt *int64   `json:"logged_at,omitempty"`
		Time     *float64 `json:"time,omitempty"`
		Step     *int     `json:"step,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	// Validate mandatory fields and collect missing ones
	var missing []string
	if req.RunUUID == "" {
		missing = append(missing, "run_uuid")
	}
	if req.Key == "" {
		missing = append(missing, "key")
	}
	if req.Value == nil {
		missing = append(missing, "value")
	}
	if req.LoggedAt == nil {
		missing = append(missing, "logged_at")
	}

	if len(missing) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":          "Missing required fields",
			"missing_fields": missing,
		})
		return
	}

	// Get run_id from uuid
	var runID int
	err := db.QueryRow("SELECT id FROM runs WHERE uuid = ?", req.RunUUID).Scan(&runID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Run not found"})
		return
	}

	// Insert metric
	_, err = db.Exec(
		"INSERT INTO metrics (run_id, key, value, logged_at, time, step) VALUES (?, ?, ?, ?, ?, ?)",
		runID, req.Key, *req.Value, *req.LoggedAt, req.Time, req.Step,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert metric"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleAPILogArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (32MB max)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse multipart form"})
		return
	}

	// Get form values
	runUUID := r.FormValue("run_uuid")
	artifactPath := r.FormValue("path")

	if runUUID == "" || artifactPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required fields: run_uuid, path"})
		return
	}

	// Get run_id from uuid
	var runID int
	err = db.QueryRow("SELECT id FROM runs WHERE uuid = ?", runUUID).Scan(&runID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Run not found"})
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Store artifact
	uri, err := storeArtifact(runUUID, artifactPath, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to store artifact: %v", err)})
		return
	}

	// Insert artifact metadata into database
	_, err = db.Exec(
		"INSERT OR REPLACE INTO artifacts (run_id, path, uri) VALUES (?, ?, ?)",
		runID, artifactPath, uri,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert artifact metadata"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"path":   artifactPath,
		"uri":    uri,
	})
}

type Parameter struct {
	Key   string
	Value string
	Type  string
}

type MetricValue struct {
	Value    string
	LoggedAt string
	Time     string
	Step     string
}

type Metric struct {
	Key    string
	Values []MetricValue
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

	// Query metrics for this run, grouped by key
	metricRows, err := db.Query(`
		SELECT key, value, logged_at, time, step
		FROM metrics
		WHERE run_id = ?
		ORDER BY key, step, time`, runID)
	if err != nil {
		log.Fatalf("Failed to query metrics: %v", err)
	}
	defer metricRows.Close()

	// Group metrics by key
	metricsMap := make(map[string][]MetricValue)
	for metricRows.Next() {
		var key string
		var value float64
		var loggedAt time.Time
		var timeVal sql.NullFloat64
		var step sql.NullInt64

		err := metricRows.Scan(&key, &value, &loggedAt, &timeVal, &step)
		if err != nil {
			log.Fatalf("Failed to scan metric: %v", err)
		}

		timeStr := ""
		if timeVal.Valid {
			timeStr = fmt.Sprintf("%g", timeVal.Float64)
		}

		stepStr := ""
		if step.Valid {
			stepStr = fmt.Sprintf("%d", step.Int64)
		}

		metricsMap[key] = append(metricsMap[key], MetricValue{
			Value:    fmt.Sprintf("%g", value),
			LoggedAt: fmt.Sprintf("%d", loggedAt.UnixMilli()),
			Time:     timeStr,
			Step:     stepStr,
		})
	}

	// Convert to slice of Metric
	var metrics []Metric
	for key, values := range metricsMap {
		metrics = append(metrics, Metric{
			Key:    key,
			Values: values,
		})
	}

	data := struct {
		Title      string
		UUID       string
		Name       string
		Parameters []Parameter
		Metrics    []Metric
	}{
		Title:      name,
		UUID:       runUUID,
		Name:       name,
		Parameters: parameters,
		Metrics:    metrics,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/header.html", "templates/run.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.ExecuteTemplate(w, "run.html", data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}
