package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func main() {
	// Parse command line flags
	dbConnString := flag.String("db", "sqlite:///apparatus.db", "Database connection string (e.g., sqlite:///path/to/db.db)")
	artifactStoreURI := flag.String("artifact-store-uri", "file://artifacts", "URI for location to store artifacts (e.g. file:///path/to/artifacts")
	flag.Parse()

	// Environment variable takes precedence over command line flag
	finalDBConnString := *dbConnString
	if envDB := os.Getenv("APPARATUS_DB_CONNECTION_STRING"); envDB != "" {
		finalDBConnString = envDB
	}

	initDB(finalDBConnString)
	initArtifactStore(*artifactStoreURI)

	// Define routes
	http.Handle("/", LoggerMiddleware(http.HandlerFunc(handleHome)))
	http.Handle("/health", LoggerMiddleware(http.HandlerFunc(handleHealth)))
	http.Handle("/api/runs", LoggerMiddleware(http.HandlerFunc(handleAPICreateRun)))
	http.Handle("/api/params", LoggerMiddleware(http.HandlerFunc(handleAPILogParam)))
	http.Handle("/api/metrics", LoggerMiddleware(http.HandlerFunc(handleAPILogMetrics)))
	http.Handle("/api/artifacts", LoggerMiddleware(http.HandlerFunc(handleAPILogArtifact)))
	http.Handle("/api/runs/notes", LoggerMiddleware(http.HandlerFunc(handleAPIUpdateRunNotes)))
	http.Handle("/api/experiments", LoggerMiddleware(http.HandlerFunc(handleAPICreateExperiment)))
	http.Handle("/experiments/", LoggerMiddleware(http.HandlerFunc(handleViewExperiment)))
	http.Handle("/runs/", LoggerMiddleware(http.HandlerFunc(handleViewRun)))
	http.Handle("/artifacts", LoggerMiddleware(http.HandlerFunc(handleViewArtifact)))
	http.Handle("/artifacts/blob", LoggerMiddleware(http.HandlerFunc(handleServeArtifactBlob)))

	// Serve static files from embedded or filesystem
	staticFS, err := fs.Sub(templateFS, "static")
	if err != nil {
		log.Fatalf("Failed to get static subdirectory: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Start server
	port := "8080"
	log.Printf("Starting Apparatus server on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

type Run struct {
	UUID         string
	Name         string
	Notes        string
	CreatedAt    string
	ParentRunID  *int
	NestingLevel int
}

// NestedRun represents a run with its children for hierarchical display
type NestedRun struct {
	Run
	ID         int
	ChildCount int
	Children   []NestedRun
}

type Experiment struct {
	UUID            string
	Name            string
	CreatedAt       string
	MostRecentRunAt string
	RunCount        int
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture the status code
		lrw := &loggingResponseWriter{ResponseWriter: w}

		// Call the next handler in the chain
		next.ServeHTTP(lrw, r)

		// Log the request and response details
		log.Printf(
			"Method: %s, Path: %s, Status: %d, Latency: %v",
			r.Method,
			r.URL.Path,
			lrw.statusCode,
			time.Since(start),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Query all experiments
	experiments, err := dao.GetAllExperiments()
	if err != nil {
		log.Fatalf("Failed to query experiments: %v", err)
	}

	data := struct {
		Title       string
		Experiments []Experiment
	}{
		Title:       "Home",
		Experiments: experiments,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(templateFS, "templates/header.html", "templates/home.html")
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
	experimentUUID := r.URL.Query().Get("experiment_uuid")
	parentRunUUID := r.URL.Query().Get("parent_run_uuid")
	runUUID := uuid.New().String()

	// Get experiment ID (use default if not specified)
	var experimentID int
	var err error
	if experimentUUID == "" {
		experimentID, err = dao.GetDefaultExperimentID()
	} else {
		experimentID, err = dao.GetExperimentIDByUUID(experimentUUID)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid experiment"})
		return
	}

	// Get parent run ID if specified
	var parentRunID *int
	if parentRunUUID != "" {
		id, err := dao.GetRunIDByUUID(parentRunUUID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid parent run"})
			return
		}
		parentRunID = &id

		// If no experiment was specified, inherit from parent
		if experimentUUID == "" {
			parentRun, err := dao.GetRunByUUID(parentRunUUID)
			if err == nil && parentRun.ParentRunID != nil {
				// Get experiment from parent (query by run ID)
				// For now, keep the default experiment since we'd need to add a method
				// to get experiment_id by run_id
			}
		}
	}

	err = dao.InsertRun(runUUID, name, experimentID, parentRunID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
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

func handleAPILogMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type MetricVal struct {
		XValue float64 `json:"x_value"`
		YValue float64 `json:"y_value"`
	}
	var req struct {
		RunUUID             string       `json:"run_uuid"`
		Key                 string       `json:"key"`
		Values              *[]MetricVal `json:"values,omitempty"`
		LoggedAtEpochMillis *int64       `json:"logged_at_epoch_millis,omitempty"`
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
        if req.Values == nil {
                missing = append(missing, "values")
        }
	if req.LoggedAtEpochMillis == nil {
		missing = append(missing, "logged_at_epoch_millis")
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

	nValues := len(*req.Values)
	xValues := make([]float64, nValues, nValues)
	yValues := make([]float64, nValues, nValues)
	for i, metricVal := range *req.Values {
		xValues[i] = metricVal.XValue
		yValues[i] = metricVal.YValue
	}

	// Insert metric
	err = dao.InsertMetrics(runID, req.Key, xValues, yValues, *req.LoggedAtEpochMillis)
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

	if err := isValidArtifactPath(artifactPath); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Invalid artifact path: %v", err)})
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

func handleAPIUpdateRunNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RunUUID string `json:"run_uuid"`
		Notes   string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.RunUUID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required field: run_uuid"})
		return
	}

	runID, err := dao.GetRunIDByUUID(req.RunUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Run not found"})
		return
	}

	err = dao.UpdateRunNotes(runID, req.Notes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update notes"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleAPICreateExperiment(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	experimentUUID := uuid.New().String()

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required field: name"})
		return
	}

	err := dao.InsertExperiment(experimentUUID, name)
	if err != nil {
		log.Printf("Failed to insert experiment: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create experiment"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   experimentUUID,
		"name": name,
	})
}

func handleViewExperiment(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/experiments/")
	experimentUUID := strings.TrimSuffix(path, "/")

	experiment, err := dao.GetExperimentByUUID(experimentUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Experiment not found")
		return
	}

	experimentID, err := dao.GetExperimentIDByUUID(experimentUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Experiment not found")
		return
	}

	// Get URL query params for open state
	openL0 := r.URL.Query().Get("open_l0")
	openL1 := r.URL.Query().Get("open_l1")

	// Get level 0 runs
	level0Runs, err := dao.GetRunsByExperimentIDAndLevel(experimentID, 0)
	if err != nil {
		log.Printf("Failed to get level 0 runs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Build nested run structure
	// TODO(a-1ebf): Fix N+1 queries - runID and childCount should come from DAO
	var nestedRuns []NestedRun
	for _, run := range level0Runs {
		runID, _ := dao.GetRunIDByUUID(run.UUID)
		childCount, _ := dao.GetChildRunCount(runID)

		nestedRun := NestedRun{
			Run:        run,
			ID:         runID,
			ChildCount: childCount,
		}

		// If this run is open, load its children
		if run.UUID == openL0 && childCount > 0 {
			childRuns, _ := dao.GetChildRuns(runID)
			for _, childRun := range childRuns {
				childRunID, _ := dao.GetRunIDByUUID(childRun.UUID)
				grandchildCount, _ := dao.GetChildRunCount(childRunID)

				childNestedRun := NestedRun{
					Run:        childRun,
					ID:         childRunID,
					ChildCount: grandchildCount,
				}

				// If this child is open, load its grandchildren
				if childRun.UUID == openL1 && grandchildCount > 0 {
					grandchildRuns, _ := dao.GetChildRuns(childRunID)
					for _, grandchildRun := range grandchildRuns {
						grandchildRunID, _ := dao.GetRunIDByUUID(grandchildRun.UUID)
						childNestedRun.Children = append(childNestedRun.Children, NestedRun{
							Run: grandchildRun,
							ID:  grandchildRunID,
						})
					}
				}

				nestedRun.Children = append(nestedRun.Children, childNestedRun)
			}
		}

		nestedRuns = append(nestedRuns, nestedRun)
	}

	data := struct {
		Title          string
		Experiment     *Experiment
		NestedRuns     []NestedRun
		OpenL0         string
		OpenL1         string
		ExperimentUUID string
	}{
		Title:          experiment.Name,
		Experiment:     experiment,
		NestedRuns:     nestedRuns,
		OpenL0:         openL0,
		OpenL1:         openL1,
		ExperimentUUID: experimentUUID,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(templateFS, "templates/header.html", "templates/experiment.html")
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "experiment.html", data)
	if err != nil {
		log.Printf("Failed to execute template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
func handleUpdateRunNotes(w http.ResponseWriter, r *http.Request, runUUID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	notes := r.FormValue("notes")

	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Run not found"})
		return
	}

	err = dao.UpdateRunNotes(runID, notes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update notes"})
		return
	}

	// Return the notes form fragment for htmx to swap in
	data := struct {
		UUID  string
		Notes string
	}{
		UUID:  runUUID,
		Notes: notes,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(templateFS, "templates/run_notes_form.html")
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "notes_form", data)
}

type Parameter struct {
	Key   string
	Value string
	Type  string
}

type MetricValue struct {
	XValue   string
	YValue   string
	LoggedAt string
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
		case "notes":
			handleUpdateRunNotes(w, r, runUUID)
			return
		}
	}

	// Main run page
	run, err := dao.GetRunByUUID(runUUID)
	if err != nil {
		log.Fatalf("Failed to query run: %v", err)
	}
	name := run.Name

	// Get parent run info if exists
	var parentRun *Run
	var grandparentRun *Run
	if run.ParentRunID != nil {
		var err error
		parentRun, err = dao.GetRunByID(*run.ParentRunID)
		if err != nil {
			log.Printf("Failed to get parent run (id=%d) for run %s: %v", *run.ParentRunID, runUUID, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Internal server error")
			return
		}
		if parentRun != nil && parentRun.ParentRunID != nil {
			grandparentRun, err = dao.GetRunByID(*parentRun.ParentRunID)
			if err != nil {
				log.Printf("Failed to get grandparent run (id=%d) for run %s: %v", *parentRun.ParentRunID, runUUID, err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Internal server error")
				return
			}
		}
	}

	// Get experiment for this run
	experiment, err := dao.GetExperimentForRunUUID(runUUID)
	if err != nil {
		log.Printf("Failed to get experiment for run %s: %v", runUUID, err)
		// Don't fail the request, just leave experiment nil
		experiment = nil
	}

	data := struct {
		Title          string
		UUID           string
		Name           string
		ParentRun      *Run
		GrandparentRun *Run
		Experiment     *Experiment
	}{
		Title:          name,
		UUID:           runUUID,
		Name:           name,
		ParentRun:      parentRun,
		GrandparentRun: grandparentRun,
		Experiment:     experiment,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(templateFS, "templates/header.html", "templates/run.html")
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
	tmpl, err := template.ParseFS(templateFS, "templates/run_page_tabs.html")
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
		metricsMap[m.Key] = append(metricsMap[m.Key], MetricValue{
			XValue:   fmt.Sprintf("%g", m.XValue),
			YValue:   fmt.Sprintf("%g", m.YValue),
			LoggedAt: fmt.Sprintf("%d", m.LoggedAt.UnixMilli()),
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
		Notes      string
		Parameters []Parameter
		Metrics    []Metric
	}{
		Title:      name,
		UUID:       runUUID,
		Name:       name,
		Notes:      run.Notes,
		Parameters: parameters,
		Metrics:    metrics,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFS(templateFS, "templates/run_overview.html", "templates/run_notes_form.html")
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
	tmpl, err = tmpl.ParseFS(templateFS, "templates/run_artifacts.html")
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
	tmpl, err := template.ParseFS(templateFS, "templates/artifact_display.html")
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
		requestedPath := strings.TrimPrefix(artifactURI, "file://")
		if filepath.IsAbs(requestedPath) {
			http.Error(w, "Forbidden absolute path", http.StatusForbidden)
		}
		cleanPath := filepath.Clean(filepath.Join(artifactStorePath, requestedPath))

		// Ensure the path is within the artifact store to prevent path traversal
		if !strings.HasPrefix(cleanPath, artifactStorePath) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		http.ServeFile(w, r, cleanPath)
		return
	} else {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}
