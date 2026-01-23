package main

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// testDAOImplementation runs a comprehensive test suite for a DAO implementation
func testDAOImplementation(t *testing.T, dao DAO) {
	// Get default experiment ID for run creation
	defaultExpID, err := dao.GetDefaultExperimentID()
	if err != nil {
		t.Fatalf("GetDefaultExperimentID failed: %v", err)
	}

	// Test InsertExperiment and GetExperimentByUUID
	expUUID := "test-exp-uuid-123"
	expName := "Test Experiment"
	err = dao.InsertExperiment(expUUID, expName)
	if err != nil {
		t.Fatalf("InsertExperiment failed: %v", err)
	}

	exp, err := dao.GetExperimentByUUID(expUUID)
	if err != nil {
		t.Fatalf("GetExperimentByUUID failed: %v", err)
	}
	if exp.UUID != expUUID || exp.Name != expName {
		t.Errorf("GetExperimentByUUID returned incorrect data: got %+v", exp)
	}

	// Test GetExperimentIDByUUID
	expID, err := dao.GetExperimentIDByUUID(expUUID)
	if err != nil {
		t.Fatalf("GetExperimentIDByUUID failed: %v", err)
	}
	if expID <= 0 {
		t.Errorf("GetExperimentIDByUUID returned invalid ID: %d", expID)
	}

	// Test GetAllExperiments includes our new experiment
	experiments, err := dao.GetAllExperiments()
	if err != nil {
		t.Fatalf("GetAllExperiments failed: %v", err)
	}
	if len(experiments) < 2 {
		t.Errorf("Expected at least 2 experiments (default + test), got %d", len(experiments))
	}

	// Test run under non-default experiment and GetRunsByExperimentID
	runUnderExpUUID := "run-under-exp-uuid"
	err = dao.InsertRun(runUnderExpUUID, "Run Under Test Experiment", expID, nil)
	if err != nil {
		t.Fatalf("InsertRun under experiment failed: %v", err)
	}

	runsForExp, err := dao.GetRunsByExperimentID(expID)
	if err != nil {
		t.Fatalf("GetRunsByExperimentID failed: %v", err)
	}
	if len(runsForExp) != 1 {
		t.Errorf("Expected 1 run for experiment, got %d", len(runsForExp))
	}
	if runsForExp[0].UUID != runUnderExpUUID {
		t.Errorf("GetRunsByExperimentID returned wrong run: expected %s, got %s", runUnderExpUUID, runsForExp[0].UUID)
	}

	// Test experiment isolation: create second experiment with a run
	exp2UUID := "test-exp-uuid-456"
	err = dao.InsertExperiment(exp2UUID, "Second Experiment")
	if err != nil {
		t.Fatalf("InsertExperiment for exp2 failed: %v", err)
	}
	exp2ID, err := dao.GetExperimentIDByUUID(exp2UUID)
	if err != nil {
		t.Fatalf("GetExperimentIDByUUID for exp2 failed: %v", err)
	}

	runUnderExp2UUID := "run-under-exp2-uuid"
	err = dao.InsertRun(runUnderExp2UUID, "Run Under Second Experiment", exp2ID, nil)
	if err != nil {
		t.Fatalf("InsertRun under exp2 failed: %v", err)
	}

	// Verify exp1 still only has 1 run
	runsForExp, err = dao.GetRunsByExperimentID(expID)
	if err != nil {
		t.Fatalf("GetRunsByExperimentID for exp1 failed: %v", err)
	}
	if len(runsForExp) != 1 {
		t.Errorf("Expected 1 run for exp1 after adding run to exp2, got %d", len(runsForExp))
	}

	// Verify exp2 has exactly 1 run
	runsForExp2, err := dao.GetRunsByExperimentID(exp2ID)
	if err != nil {
		t.Fatalf("GetRunsByExperimentID for exp2 failed: %v", err)
	}
	if len(runsForExp2) != 1 {
		t.Errorf("Expected 1 run for exp2, got %d", len(runsForExp2))
	}
	if runsForExp2[0].UUID != runUnderExp2UUID {
		t.Errorf("GetRunsByExperimentID for exp2 returned wrong run: expected %s, got %s", runUnderExp2UUID, runsForExp2[0].UUID)
	}

	// Test InsertRun and GetRunByUUID
	runUUID := "test-run-uuid-123"
	runName := "Test Run"
	err = dao.InsertRun(runUUID, runName, defaultExpID, nil)
	if err != nil {
		t.Fatalf("InsertRun failed: %v", err)
	}

	run, err := dao.GetRunByUUID(runUUID)
	if err != nil {
		t.Fatalf("GetRunByUUID failed: %v", err)
	}
	if run.UUID != runUUID || run.Name != runName {
		t.Errorf("GetRunByUUID returned incorrect data: got %+v", run)
	}

	// Test GetRunIDByUUID
	runID, err := dao.GetRunIDByUUID(runUUID)
	if err != nil {
		t.Fatalf("GetRunIDByUUID failed: %v", err)
	}
	if runID <= 0 {
		t.Errorf("GetRunIDByUUID returned invalid ID: %d", runID)
	}

	// Test GetAllRuns
	runs, err := dao.GetAllRuns()
	if err != nil {
		t.Fatalf("GetAllRuns failed: %v", err)
	}
	if len(runs) == 0 {
		t.Error("GetAllRuns returned no runs")
	}

	// Test UpsertParameter with different types
	testCases := []struct {
		key         string
		valueType   string
		valueString *string
		valueBool   *bool
		valueFloat  *float64
		valueInt    *int64
	}{
		{
			key:        "learning_rate",
			valueType:  "float",
			valueFloat: floatPtr(0.001),
		},
		{
			key:       "epochs",
			valueType: "int",
			valueInt:  int64Ptr(100),
		},
		{
			key:         "model_name",
			valueType:   "string",
			valueString: stringPtr("bert-base"),
		},
		{
			key:       "use_gpu",
			valueType: "bool",
			valueBool: &[]bool{true}[0],
		},
	}

	for _, tc := range testCases {
		err = dao.UpsertParameter(runID, tc.key, tc.valueType, tc.valueString, tc.valueBool, tc.valueFloat, tc.valueInt)
		if err != nil {
			t.Fatalf("UpsertParameter failed for %s: %v", tc.key, err)
		}
	}

	// Test GetParametersByRunID
	params, err := dao.GetParametersByRunID(runID)
	if err != nil {
		t.Fatalf("GetParametersByRunID failed: %v", err)
	}
	if len(params) != len(testCases) {
		t.Errorf("Expected %d parameters, got %d", len(testCases), len(params))
	}

	// Test InsertMetric
	now := time.Now()
	err = dao.InsertMetrics(runID, "loss", []float64{0, 10, 20, 30},
		[]float64{0.5, 0.37, 0.34, 0.21}, now.UnixMilli())
	if err != nil {
		t.Fatalf("InsertMetric failed: %v", err)
	}

	// Test GetMetricsByRunID
	metrics, err := dao.GetMetricsByRunID(runID)
	if err != nil {
		t.Fatalf("GetMetricsByRunID failed: %v", err)
	}
	if len(metrics) != 4 {
		t.Errorf("Expected 4 metrics, got %d", len(metrics))
	}
	if metrics[2].XValue != 20.0 {
		t.Errorf("XValue was incorrect")
	}
	if metrics[1].YValue != 0.37 {
		t.Errorf("YValue was incorrect")
	}
	if metrics[1].LoggedAt.UnixMilli() != now.UnixMilli() {
		t.Errorf("LoggedAt incorrect. Expected %v, was %v",
			now.UnixMilli(), metrics[1].LoggedAt.UnixMilli())
	}

	// Test UpsertArtifact
	err = dao.UpsertArtifact(runID, "model.pkl", "file:///path/to/model.pkl", "model")
	if err != nil {
		t.Fatalf("UpsertArtifact failed: %v", err)
	}

	err = dao.UpsertArtifact(runID, "plot.png", "file:///path/to/plot.png", "image")
	if err != nil {
		t.Fatalf("UpsertArtifact failed: %v", err)
	}

	// Test GetArtifactsByRunID
	artifacts, err := dao.GetArtifactsByRunID(runID)
	if err != nil {
		t.Fatalf("GetArtifactsByRunID failed: %v", err)
	}
	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}

	// Test GetArtifactByRunIDAndPath
	artifact, err := dao.GetArtifactByRunIDAndPath(runID, "model.pkl")
	if err != nil {
		t.Fatalf("GetArtifactByRunIDAndPath failed: %v", err)
	}
	if artifact.Path != "model.pkl" || artifact.URI != "file:///path/to/model.pkl" || artifact.Type != "model" {
		t.Errorf("GetArtifactByRunIDAndPath returned incorrect data: got %+v", artifact)
	}

	// Test upsert behavior - update existing parameter
	newFloatValue := 0.002
	err = dao.UpsertParameter(runID, "learning_rate", "float", nil, nil, &newFloatValue, nil)
	if err != nil {
		t.Fatalf("UpsertParameter update failed: %v", err)
	}

	params, err = dao.GetParametersByRunID(runID)
	if err != nil {
		t.Fatalf("GetParametersByRunID failed after update: %v", err)
	}

	// Find the learning_rate parameter and check it was updated
	found := false
	for _, p := range params {
		if p.Key == "learning_rate" {
			if !p.ValueFloat.Valid || p.ValueFloat.Float64 != newFloatValue {
				t.Errorf("Parameter not updated correctly: expected %f, got %f", newFloatValue, p.ValueFloat.Float64)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("learning_rate parameter not found after update")
	}
}

func TestSQLiteDAO(t *testing.T) {
	// Create a temporary database file with absolute path
	dbFile := "test_sqlite.db"
	defer os.Remove(dbFile)

	// Get absolute path
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	absDBPath := absPath + "/" + dbFile

	// Create the database file first by opening it
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}
	// Ping to ensure the file is created
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
	db.Close()

	connString := "sqlite:///" + absDBPath

	// Run migrations
	m, err := migrate.New("file://migrations/sqlite3", connString)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Reopen database connection
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("Failed to reopen SQLite database: %v", err)
	}
	defer db.Close()

	dao := NewSQLiteDAO(db)
	testDAOImplementation(t, dao)
}

func TestPostgresDAO(t *testing.T) {
	// Skip if no Postgres connection string is provided
	connString := os.Getenv("POSTGRES_TEST_DB")
	if connString == "" {
		t.Error("POSTGRES_TEST_DB environment variable not set")
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("Failed to open Postgres database: %v", err)
	}
	defer db.Close()

	// Drop and recreate the schema using migrations
	m, err := migrate.New("file://migrations/postgres", connString)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	// Drop all tables
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Logf("Warning during migration down: %v", err)
	}
	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	dao := NewPostgresDAO(db)
	testDAOImplementation(t, dao)
}

// Helper functions to create pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}
