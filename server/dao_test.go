package main

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// testDAOImplementation runs a comprehensive test suite for a DAO implementation
func testDAOImplementation(t *testing.T, dao DAO) {
	// Test CreateTables
	err := dao.CreateTables()
	if err != nil {
		t.Fatalf("CreateTables failed: %v", err)
	}

	// Test InsertRun and GetRunByUUID
	runUUID := "test-run-uuid-123"
	runName := "Test Run"
	err = dao.InsertRun(runUUID, runName)
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
	now := time.Now().UnixMilli()
	err = dao.InsertMetric(runID, "loss", 0.5, now, floatPtr(1.0), intPtr(1))
	if err != nil {
		t.Fatalf("InsertMetric failed: %v", err)
	}

	err = dao.InsertMetric(runID, "loss", 0.3, now+1000, floatPtr(2.0), intPtr(2))
	if err != nil {
		t.Fatalf("InsertMetric failed: %v", err)
	}

	err = dao.InsertMetric(runID, "accuracy", 0.9, now, floatPtr(1.0), intPtr(1))
	if err != nil {
		t.Fatalf("InsertMetric failed: %v", err)
	}

	// Test GetMetricsByRunID
	metrics, err := dao.GetMetricsByRunID(runID)
	if err != nil {
		t.Fatalf("GetMetricsByRunID failed: %v", err)
	}
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
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
	// Create a temporary database file
	dbFile := "test_sqlite.db"
	defer os.Remove(dbFile)

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
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

	// Clean up tables before test
	cleanupSQL := []string{
		"DROP TABLE IF EXISTS artifacts",
		"DROP TABLE IF EXISTS metrics",
		"DROP TABLE IF EXISTS parameters",
		"DROP TABLE IF EXISTS runs",
	}
	for _, sql := range cleanupSQL {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("Failed to clean up tables: %v", err)
		}
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
