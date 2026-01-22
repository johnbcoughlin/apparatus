package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQLiteDAO implements the DAO interface for SQLite
type SQLiteDAO struct {
	db *sql.DB
}

// NewSQLiteDAO creates a new SQLite DAO
func NewSQLiteDAO(db *sql.DB) *SQLiteDAO {
	return &SQLiteDAO{db: db}
}

// InsertRun inserts a new run
func (d *SQLiteDAO) InsertRun(uuid, name string) error {
	_, err := d.db.Exec(
		"INSERT INTO runs (uuid, name) VALUES (?, ?)",
		uuid, name,
	)
	return err
}

// GetRunByUUID retrieves a run by its UUID
func (d *SQLiteDAO) GetRunByUUID(uuid string) (*Run, error) {
	var name, notes string
	err := d.db.QueryRow(
		"SELECT name, notes FROM runs WHERE uuid = ?",
		uuid,
	).Scan(&name, &notes)
	if err != nil {
		return nil, err
	}
	return &Run{UUID: uuid, Name: name, Notes: notes}, nil
}

// GetRunIDByUUID retrieves the database ID of a run by its UUID
func (d *SQLiteDAO) GetRunIDByUUID(uuid string) (int, error) {
	var id int
	err := d.db.QueryRow(
		"SELECT id FROM runs WHERE uuid = ?",
		uuid,
	).Scan(&id)
	return id, err
}

// GetAllRuns retrieves all runs ordered by created_at descending
func (d *SQLiteDAO) GetAllRuns() ([]Run, error) {
	rows, err := d.db.Query(`
		SELECT uuid, name, created_at
		FROM runs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var uuid, name, createdAt string
		if err := rows.Scan(&uuid, &name, &createdAt); err != nil {
			return nil, err
		}
		runs = append(runs, Run{UUID: uuid, Name: name, CreatedAt: createdAt})
	}

	return runs, rows.Err()
}

// UpsertParameter inserts or updates a parameter
func (d *SQLiteDAO) UpsertParameter(runID int, key, valueType string, valueString *string, valueBool *bool, valueFloat *float64, valueInt *int64) error {
	var sql string
	var args []interface{}

	switch valueType {
	case "string":
		sql = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_string) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, valueString}
	case "bool":
		sql = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_bool) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, valueBool}
	case "float":
		sql = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_float) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, valueFloat}
	case "int":
		sql = "INSERT OR REPLACE INTO parameters (run_id, key, value_type, value_int) VALUES (?, ?, ?, ?)"
		args = []interface{}{runID, key, valueType, valueInt}
	default:
		return fmt.Errorf("unsupported value type: %s", valueType)
	}

	_, err := d.db.Exec(sql, args...)
	return err
}

// GetParametersByRunID retrieves all parameters for a run
func (d *SQLiteDAO) GetParametersByRunID(runID int) ([]ParameterRow, error) {
	rows, err := d.db.Query(`
		SELECT key, value_type, value_string, value_bool, value_float, value_int
		FROM parameters
		WHERE run_id = ?
		ORDER BY key
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []ParameterRow
	for rows.Next() {
		var p ParameterRow
		if err := rows.Scan(&p.Key, &p.ValueType, &p.ValueString, &p.ValueBool, &p.ValueFloat, &p.ValueInt); err != nil {
			return nil, err
		}
		params = append(params, p)
	}

	return params, rows.Err()
}

// InsertMetric inserts a new metric
func (d *SQLiteDAO) InsertMetrics(runID int, key string, xValues []float64, yValues []float64, loggedAtEpochMillis int64) error {
	if len(xValues) != len(yValues) {
		return errors.New("xValues and yValues must have the same length")
	}
	var stmtBuilder strings.Builder
	stmtBuilder.WriteString("INSERT INTO metrics (run_id, key, x_value, y_value, logged_at) VALUES")
	vals := []interface{}{}
	for i := range len(xValues) {
		stmtBuilder.WriteString("(?, ?, ?, ?, ?)")
		if i < len(xValues)-1 {
			stmtBuilder.WriteString(", ")
		}
		vals = append(vals, runID, key, xValues[i], yValues[i], time.UnixMilli(loggedAtEpochMillis).UTC())
	}
	stmtBuilder.WriteString(";")
	stmt, err := d.db.Prepare(stmtBuilder.String())
	if err != nil {
		return err
	}
	_, err = stmt.Exec(vals...)
	return err
}

// GetMetricsByRunID retrieves all metrics for a run
func (d *SQLiteDAO) GetMetricsByRunID(runID int) ([]MetricRow, error) {
	rows, err := d.db.Query(`
		SELECT key, x_value, y_value, logged_at
		FROM metrics
		WHERE run_id = ?
		ORDER BY key, x_value
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []MetricRow
	for rows.Next() {
		var m MetricRow
		if err := rows.Scan(&m.Key, &m.XValue, &m.YValue, &m.LoggedAt); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// UpsertArtifact inserts or updates an artifact
func (d *SQLiteDAO) UpsertArtifact(runID int, path, uri, artifactType string) error {
	_, err := d.db.Exec(
		"INSERT OR REPLACE INTO artifacts (run_id, path, uri, type) VALUES (?, ?, ?, ?)",
		runID, path, uri, artifactType,
	)
	return err
}

// GetArtifactsByRunID retrieves all artifacts for a run
func (d *SQLiteDAO) GetArtifactsByRunID(runID int) ([]ArtifactRow, error) {
	rows, err := d.db.Query(`
		SELECT path, uri, type
		FROM artifacts
		WHERE run_id = ?
		ORDER BY path
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []ArtifactRow
	for rows.Next() {
		var a ArtifactRow
		if err := rows.Scan(&a.Path, &a.URI, &a.Type); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, a)
	}

	return artifacts, rows.Err()
}

// GetArtifactByRunIDAndPath retrieves a specific artifact by run ID and path
func (d *SQLiteDAO) GetArtifactByRunIDAndPath(runID int, path string) (*ArtifactRow, error) {
	var a ArtifactRow
	err := d.db.QueryRow(
		"SELECT path, uri, type FROM artifacts WHERE run_id = ? AND path = ?",
		runID, path,
	).Scan(&a.Path, &a.URI, &a.Type)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateRunNotes updates the notes for a run
func (d *SQLiteDAO) UpdateRunNotes(runID int, notes string) error {
	_, err := d.db.Exec(
		"UPDATE runs SET notes = ? WHERE id = ?",
		notes, runID,
	)
	return err
}
