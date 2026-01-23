package main

import (
	"database/sql"
	"time"
)

// DAO defines the interface for database operations
type DAO interface {
	// Experiment operations
	InsertExperiment(uuid, name string) error
	GetExperimentByUUID(uuid string) (*Experiment, error)
	GetExperimentIDByUUID(uuid string) (int, error)
	GetAllExperiments() ([]Experiment, error)
	GetDefaultExperimentID() (int, error)

	// Run operations
	InsertRun(uuid, name string, experimentID int) error
	GetRunByUUID(uuid string) (*Run, error)
	GetRunIDByUUID(uuid string) (int, error)
	GetAllRuns() ([]Run, error)
	GetRunsByExperimentID(experimentID int) ([]Run, error)
	UpdateRunNotes(runID int, notes string) error

	// Parameter operations
	UpsertParameter(runID int, key, valueType string, valueString *string, valueBool *bool, valueFloat *float64, valueInt *int64) error
	GetParametersByRunID(runID int) ([]ParameterRow, error)

	// Metric operations
	InsertMetrics(runID int, key string, xValues []float64, yValues []float64, loggedAt int64) error
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
	Notes     string
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
	XValue   float64
	YValue   float64
	LoggedAt time.Time
}

// ArtifactRow represents a row in the artifacts table
type ArtifactRow struct {
	Path string
	URI  string
	Type string
}

// ExperimentRow represents a row in the experiments table
type ExperimentRow struct {
	ID        int
	UUID      string
	Name      string
	CreatedAt time.Time
}
