package main

import (
	"database/sql"
	"time"
)

// DAO defines the interface for database operations
type DAO interface {
	// Schema operations
	CreateTables() error

	// Run operations
	InsertRun(uuid, name string) error
	GetRunByUUID(uuid string) (*Run, error)
	GetRunIDByUUID(uuid string) (int, error)
	GetAllRuns() ([]Run, error)

	// Parameter operations
	UpsertParameter(runID int, key, valueType string, valueString *string, valueBool *bool, valueFloat *float64, valueInt *int64) error
	GetParametersByRunID(runID int) ([]ParameterRow, error)

	// Metric operations
	InsertMetric(runID int, key string, value float64, loggedAt int64, time *float64, step *int) error
	GetMetricsByRunID(runID int) ([]MetricRow, error)

	// Artifact operations
	UpsertArtifact(runID int, path, uri, artifactType string) error
	GetArtifactsByRunID(runID int) ([]ArtifactRow, error)
	GetArtifactByRunIDAndPath(runID int, path string) (*ArtifactRow, error)
}

// RunRow represents a row in the runs table
type RunRow struct {
	ID        int
	UUID      string
	Name      string
	CreatedAt time.Time
}

// ParameterRow represents a row in the parameters table
type ParameterRow struct {
	Key         string
	ValueType   string
	ValueString sql.NullString
	ValueBool   sql.NullBool
	ValueFloat  sql.NullFloat64
	ValueInt    sql.NullInt64
}

// MetricRow represents a row in the metrics table
type MetricRow struct {
	Key      string
	Value    float64
	LoggedAt time.Time
	Time     sql.NullFloat64
	Step     sql.NullInt64
}

// ArtifactRow represents a row in the artifacts table
type ArtifactRow struct {
	Path string
	URI  string
	Type string
}
