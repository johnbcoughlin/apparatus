package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

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
	http.HandleFunc("/artifacts", handleViewArtifact)
	http.HandleFunc("/artifacts/blob", handleServeArtifactBlob)
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
	runs, err := dao.GetAllRuns()
	if err != nil {
		log.Fatalf("Failed to query runs: %v", err)
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

	err := dao.InsertRun(runUUID, name)
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
	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Insert parameter based on type
	var valueString *string
	var valueBool *bool
	var valueFloat *float64
	var valueInt *int64

	switch valueType {
	case "string":
		valueString = &value
	case "bool":
		boolVal := value == "true"
		valueBool = &boolVal
	case "float":
		var f float64
		fmt.Sscanf(value, "%f", &f)
		valueFloat = &f
	case "int":
		var i int64
		fmt.Sscanf(value, "%d", &i)
		valueInt = &i
	}

	err = dao.UpsertParameter(runID, key, valueType, valueString, valueBool, valueFloat, valueInt)
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
	runID, err := dao.GetRunIDByUUID(req.RunUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Run not found"})
		return
	}

	// Insert metric
	err = dao.InsertMetric(runID, req.Key, *req.Value, *req.LoggedAt, req.Time, req.Step)
	if err != nil {
		log.Printf("Error inserting metric: %v", err)
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
	runID, err := dao.GetRunIDByUUID(runUUID)
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

	var artifactType string
	if strings.HasSuffix(artifactPath, ".png") {
		artifactType = "image"
	} else {
		artifactType = "unknown"
	}

	// Insert artifact metadata into database
	err = dao.UpsertArtifact(runID, artifactPath, uri, artifactType)
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

type Artifact struct {
	Path string
	URI  string
	Type string
}

func handleViewRun(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/runs/")
	parts := strings.SplitN(path, "/", 2)
	runUUID := parts[0]

	// Route to sub-handlers
	if len(parts) == 2 {
		switch parts[1] {
		case "overview":
			executeRunPageTabsTemplate(w, r, runUUID, "overview")
			handleRunOverview(w, r, runUUID)
			return
		case "artifacts":
			executeRunPageTabsTemplate(w, r, runUUID, "artifacts")
			handleRunArtifacts(w, r, runUUID)
			return
		}
	}

	// Main run page
	run, err := dao.GetRunByUUID(runUUID)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}
	name := run.Name

	data := struct {
		Title string
		UUID  string
		Name  string
	}{
		Title: name,
		UUID:  runUUID,
		Name:  name,
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

func executeRunPageTabsTemplate(w http.ResponseWriter, r *http.Request, runUUID string, pageName string) {
	maybeCurrentArtifactPath := r.URL.Query().Get("current_artifact_path")
	var currentArtifactPath *string
	if maybeCurrentArtifactPath == "" {
		currentArtifactPath = nil
	} else {
		currentArtifactPath = &maybeCurrentArtifactPath
	}

	data := struct {
		CurrentArtifactPath *string
		UUID                string
		PageName            string
	}{
		CurrentArtifactPath: currentArtifactPath,
		UUID:                runUUID,
		PageName:            pageName,
	}
	tmpl, err := template.ParseFiles("templates/run_page_tabs.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

func handleRunOverview(w http.ResponseWriter, r *http.Request, runUUID string) {
	run, err := dao.GetRunByUUID(runUUID)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}
	name := run.Name

	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		log.Fatalf("Failed to get run ID: %v", err)
	}

	// Query parameters for this run
	paramRows, err := dao.GetParametersByRunID(runID)
	if err != nil {
		log.Fatalf("Failed to query parameters: %v", err)
	}

	var parameters []Parameter
	for _, p := range paramRows {
		var value string
		switch p.ValueType {
		case "string":
			value = p.ValueString.String
		case "bool":
			if p.ValueBool.Bool {
				value = "true"
			} else {
				value = "false"
			}
		case "float":
			value = fmt.Sprintf("%g", p.ValueFloat.Float64)
		case "int":
			value = fmt.Sprintf("%d", p.ValueInt.Int64)
		}

		parameters = append(parameters, Parameter{Key: p.Key, Value: value, Type: p.ValueType})
	}

	// Query metrics for this run
	metricRows, err := dao.GetMetricsByRunID(runID)
	if err != nil {
		log.Fatalf("Failed to query metrics: %v", err)
	}

	// Group metrics by key
	metricsMap := make(map[string][]MetricValue)
	for _, m := range metricRows {
		timeStr := ""
		if m.Time.Valid {
			timeStr = fmt.Sprintf("%g", m.Time.Float64)
		}

		stepStr := ""
		if m.Step.Valid {
			stepStr = fmt.Sprintf("%d", m.Step.Int64)
		}

		metricsMap[m.Key] = append(metricsMap[m.Key], MetricValue{
			Value:    fmt.Sprintf("%g", m.Value),
			LoggedAt: fmt.Sprintf("%d", m.LoggedAt.UnixMilli()),
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
	tmpl, err := template.ParseFiles("templates/run_overview.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

type ArtifactsTreeNode struct {
	Children     map[string]*ArtifactsTreeNode
	ArtifactURI  *string
	ArtifactPath *string
	RunUUID      *string
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for shorter ID
}

func handleRunArtifacts(w http.ResponseWriter, r *http.Request, runUUID string) {
	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}

	// Query artifacts for this run
	artifactRows, err := dao.GetArtifactsByRunID(runID)
	if err != nil {
		log.Fatalf("Failed to query artifacts: %v", err)
	}

	var artifacts []Artifact
	for _, a := range artifactRows {
		artifacts = append(artifacts, Artifact{Path: a.Path, URI: a.URI, Type: a.Type})
	}

	artifactsTree := assembleArtifactsTree(runUUID, artifacts)

	// Pull out the current artifact for display if it's present in the request
	currentArtifactPath := r.URL.Query().Get("current_artifact_path")
	log.Println("current artifact:", currentArtifactPath)

	var currentArtifact *Artifact = nil
	if currentArtifactPath != "" {
		for _, artifact := range artifacts {
			if artifact.Path == currentArtifactPath {
				currentArtifact = &artifact
			}
		}
	}

	data := struct {
		UUID            string
		ArtifactsTree   ArtifactsTreeNode
		CurrentArtifact *Artifact
	}{
		UUID:            runUUID,
		ArtifactsTree:   artifactsTree,
		CurrentArtifact: currentArtifact,
	}

	tmpl := template.New("run_artifacts.html").Funcs(template.FuncMap{
		"hash": hashString,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err = tmpl.ParseFiles("templates/run_artifacts.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

func assembleArtifactsTree(runUUID string, artifacts []Artifact) ArtifactsTreeNode {
	root := ArtifactsTreeNode{make(map[string]*ArtifactsTreeNode), nil, nil, nil}
	for _, artifact := range artifacts {
		node := &root
		parts := strings.Split(artifact.Path, "/")
		for _, part := range parts {
			child, ok := node.Children[part]
			if ok {
				node = child
			} else {
				newNode := ArtifactsTreeNode{make(map[string]*ArtifactsTreeNode), nil, nil, nil}
				node.Children[part] = &newNode
				node = &newNode
			}
		}
		node.ArtifactURI = &artifact.URI
		node.ArtifactPath = &artifact.Path
		node.RunUUID = &runUUID
	}
	return root
}

func handleViewArtifact(w http.ResponseWriter, r *http.Request) {
	runUUID := r.URL.Query().Get("run_uuid")
	artifactPath := r.URL.Query().Get("path")

	if runUUID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Missing required parameter: run_uuid")
		return
	}

	// Get run_id from uuid
	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Run not found")
		return
	}

	// Query artifact URI and type from database
	artifact, err := dao.GetArtifactByRunIDAndPath(runID, artifactPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Artifact not found")
		return
	}

	// Render template with artifact URI and type
	data := struct {
		ArtifactURI  string
		ArtifactType string
	}{
		ArtifactURI:  artifact.URI,
		ArtifactType: artifact.Type,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles("templates/artifact_display.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

func handleServeArtifactBlob(w http.ResponseWriter, r *http.Request) {
	artifactURI := r.URL.Query().Get("uri")
	if strings.HasPrefix(artifactURI, "file://") {
		http.ServeFile(w, r, strings.TrimPrefix(artifactURI, "file://"))
		return
	}
}
