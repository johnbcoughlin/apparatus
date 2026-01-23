package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log"
	"time"
)

// PostgresDAO implements the DAO interface for PostgreSQL
type PostgresDAO struct {
	db *sql.DB
}

// NewPostgresDAO creates a new Postgres DAO
func NewPostgresDAO(db *sql.DB) *PostgresDAO {
	return &PostgresDAO{db: db}
}

// InsertExperiment inserts a new experiment
func (d *PostgresDAO) InsertExperiment(uuid, name string) error {
	_, err := d.db.Exec(
		"INSERT INTO experiments (uuid, name) VALUES ($1, $2)",
		uuid, name,
	)
	return err
}

// GetExperimentByUUID retrieves an experiment by its UUID
func (d *PostgresDAO) GetExperimentByUUID(uuid string) (*Experiment, error) {
	var name, createdAt string
	var mostRecentRunAt sql.NullString
	err := d.db.QueryRow(`
		SELECT e.name, e.created_at,
			(SELECT MAX(created_at) FROM runs WHERE experiment_id = e.id) as most_recent_run_at
		FROM experiments e WHERE e.uuid = $1`,
		uuid,
	).Scan(&name, &createdAt, &mostRecentRunAt)
	if err != nil {
		return nil, err
	}
	exp := &Experiment{UUID: uuid, Name: name, CreatedAt: createdAt}
	if mostRecentRunAt.Valid {
		exp.MostRecentRunAt = mostRecentRunAt.String
	}
	return exp, nil
}

// GetExperimentIDByUUID retrieves the database ID of an experiment by its UUID
func (d *PostgresDAO) GetExperimentIDByUUID(uuid string) (int, error) {
	var id int
	err := d.db.QueryRow(
		"SELECT id FROM experiments WHERE uuid = $1",
		uuid,
	).Scan(&id)
	return id, err
}

// GetAllExperiments retrieves all experiments ordered by most_recent_run_at descending
func (d *PostgresDAO) GetAllExperiments() ([]Experiment, error) {
	rows, err := d.db.Query(`
		SELECT e.uuid, e.name, e.created_at,
			(SELECT MAX(created_at) FROM runs WHERE experiment_id = e.id) as most_recent_run_at,
			(SELECT COUNT(*) FROM runs WHERE experiment_id = e.id) as run_count
		FROM experiments e
		ORDER BY COALESCE((SELECT MAX(created_at) FROM runs WHERE experiment_id = e.id), e.created_at) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []Experiment
	for rows.Next() {
		var uuid, name, createdAt string
		var mostRecentRunAt sql.NullString
		var runCount int
		if err := rows.Scan(&uuid, &name, &createdAt, &mostRecentRunAt, &runCount); err != nil {
			return nil, err
		}
		exp := Experiment{UUID: uuid, Name: name, CreatedAt: createdAt, RunCount: runCount}
		if mostRecentRunAt.Valid {
			exp.MostRecentRunAt = mostRecentRunAt.String
		}
		experiments = append(experiments, exp)
	}

	return experiments, rows.Err()
}

// GetDefaultExperimentID returns the ID of the default experiment
func (d *PostgresDAO) GetDefaultExperimentID() (int, error) {
	var id int
	err := d.db.QueryRow("SELECT id FROM experiments WHERE uuid = '00000000-0000-0000-0000-000000000000'").Scan(&id)
	return id, err
}

// InsertRun inserts a new run
func (d *PostgresDAO) InsertRun(uuid, name string, experimentID int, parentRunID *int) error {
	var nestingLevel int
	if parentRunID != nil {
		// Get parent's nesting level and add 1
		var parentLevel int
		err := d.db.QueryRow("SELECT nesting_level FROM runs WHERE id = $1", *parentRunID).Scan(&parentLevel)
		if err != nil {
			return fmt.Errorf("failed to get parent run nesting level: %w", err)
		}
		nestingLevel = parentLevel + 1
		if nestingLevel > 2 {
			return fmt.Errorf("maximum nesting level (2) exceeded")
		}
	}

	_, err := d.db.Exec(
		"INSERT INTO runs (uuid, name, experiment_id, parent_run_id, nesting_level) VALUES ($1, $2, $3, $4, $5)",
		uuid, name, experimentID, parentRunID, nestingLevel,
	)
	return err
}

// GetRunByUUID retrieves a run by its UUID
func (d *PostgresDAO) GetRunByUUID(uuid string) (*Run, error) {
	var name, notes string
	var parentRunID sql.NullInt64
	var nestingLevel int
	err := d.db.QueryRow(
		"SELECT name, notes, parent_run_id, nesting_level FROM runs WHERE uuid = $1",
		uuid,
	).Scan(&name, &notes, &parentRunID, &nestingLevel)
	if err != nil {
		return nil, err
	}
	run := &Run{UUID: uuid, Name: name, Notes: notes, NestingLevel: nestingLevel}
	if parentRunID.Valid {
		id := int(parentRunID.Int64)
		run.ParentRunID = &id
	}
	return run, nil
}

// GetRunIDByUUID retrieves the database ID of a run by its UUID
func (d *PostgresDAO) GetRunIDByUUID(uuid string) (int, error) {
	var id int
	err := d.db.QueryRow(
		"SELECT id FROM runs WHERE uuid = $1",
		uuid,
	).Scan(&id)
	return id, err
}

// GetAllRuns retrieves all runs ordered by created_at descending
func (d *PostgresDAO) GetAllRuns() ([]Run, error) {
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

// GetRunsByExperimentID retrieves all runs for an experiment
func (d *PostgresDAO) GetRunsByExperimentID(experimentID int) ([]Run, error) {
	rows, err := d.db.Query(`
		SELECT uuid, name, created_at, parent_run_id, nesting_level
		FROM runs
		WHERE experiment_id = $1
		ORDER BY created_at DESC
	`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var uuid, name, createdAt string
		var parentRunID sql.NullInt64
		var nestingLevel int
		if err := rows.Scan(&uuid, &name, &createdAt, &parentRunID, &nestingLevel); err != nil {
			return nil, err
		}
		run := Run{UUID: uuid, Name: name, CreatedAt: createdAt, NestingLevel: nestingLevel}
		if parentRunID.Valid {
			id := int(parentRunID.Int64)
			run.ParentRunID = &id
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetRunsByExperimentIDAndLevel retrieves runs for an experiment at a specific nesting level
func (d *PostgresDAO) GetRunsByExperimentIDAndLevel(experimentID int, nestingLevel int) ([]Run, error) {
	rows, err := d.db.Query(`
		SELECT uuid, name, created_at, parent_run_id, nesting_level
		FROM runs
		WHERE experiment_id = $1 AND nesting_level = $2
		ORDER BY created_at DESC
	`, experimentID, nestingLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var uuid, name, createdAt string
		var parentRunID sql.NullInt64
		var level int
		if err := rows.Scan(&uuid, &name, &createdAt, &parentRunID, &level); err != nil {
			return nil, err
		}
		run := Run{UUID: uuid, Name: name, CreatedAt: createdAt, NestingLevel: level}
		if parentRunID.Valid {
			id := int(parentRunID.Int64)
			run.ParentRunID = &id
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetChildRuns retrieves all direct child runs of a parent run
func (d *PostgresDAO) GetChildRuns(parentRunID int) ([]Run, error) {
	rows, err := d.db.Query(`
		SELECT uuid, name, created_at, parent_run_id, nesting_level
		FROM runs
		WHERE parent_run_id = $1
		ORDER BY created_at DESC
	`, parentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var uuid, name, createdAt string
		var pRunID sql.NullInt64
		var nestingLevel int
		if err := rows.Scan(&uuid, &name, &createdAt, &pRunID, &nestingLevel); err != nil {
			return nil, err
		}
		run := Run{UUID: uuid, Name: name, CreatedAt: createdAt, NestingLevel: nestingLevel}
		if pRunID.Valid {
			id := int(pRunID.Int64)
			run.ParentRunID = &id
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetChildRunCount returns the count of direct child runs
func (d *PostgresDAO) GetChildRunCount(parentRunID int) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM runs WHERE parent_run_id = $1", parentRunID).Scan(&count)
	return count, err
}

// UpsertParameter inserts or updates a parameter
func (d *PostgresDAO) UpsertParameter(runID int, key, valueType string, valueString *string, valueBool *bool, valueFloat *float64, valueInt *int64) error {
	var sql string
	var args []interface{}

	switch valueType {
	case "string":
		sql = `INSERT INTO parameters (run_id, key, value_type, value_string)
		       VALUES ($1, $2, $3, $4)
		       ON CONFLICT (run_id, key) DO UPDATE
		       SET value_type = EXCLUDED.value_type, value_string = EXCLUDED.value_string`
		args = []interface{}{runID, key, valueType, valueString}
	case "bool":
		sql = `INSERT INTO parameters (run_id, key, value_type, value_bool)
		       VALUES ($1, $2, $3, $4)
		       ON CONFLICT (run_id, key) DO UPDATE
		       SET value_type = EXCLUDED.value_type, value_bool = EXCLUDED.value_bool`
		args = []interface{}{runID, key, valueType, valueBool}
	case "float":
		sql = `INSERT INTO parameters (run_id, key, value_type, value_float)
		       VALUES ($1, $2, $3, $4)
		       ON CONFLICT (run_id, key) DO UPDATE
		       SET value_type = EXCLUDED.value_type, value_float = EXCLUDED.value_float`
		args = []interface{}{runID, key, valueType, valueFloat}
	case "int":
		sql = `INSERT INTO parameters (run_id, key, value_type, value_int)
		       VALUES ($1, $2, $3, $4)
		       ON CONFLICT (run_id, key) DO UPDATE
		       SET value_type = EXCLUDED.value_type, value_int = EXCLUDED.value_int`
		args = []interface{}{runID, key, valueType, valueInt}
	default:
		return fmt.Errorf("unsupported value type: %s", valueType)
	}

	_, err := d.db.Exec(sql, args...)
	return err
}

// GetParametersByRunID retrieves all parameters for a run
func (d *PostgresDAO) GetParametersByRunID(runID int) ([]ParameterRow, error) {
	rows, err := d.db.Query(`
		SELECT key, value_type, value_string, value_bool, value_float, value_int
		FROM parameters
		WHERE run_id = $1
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
func (d *PostgresDAO) InsertMetrics(runID int, key string, xValues []float64, yValues []float64, loggedAtEpochMillis int64) error {
	if len(xValues) != len(yValues) {
		return errors.New("xValues and yValues must have the same length")
	}

	txn, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("metrics", "run_id", "key", "logged_at", "x_value", "y_value"))
	if err != nil {
		return err
	}

	for i := range len(xValues) {
		stmt.Exec(runID, key, time.UnixMilli(loggedAtEpochMillis).UTC(),
			xValues[i], yValues[i])
		if err != nil {
			log.Printf("Error inserting metric: %v", err)
			return err
		}
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	return err
}

// GetMetricsByRunID retrieves all metrics for a run
func (d *PostgresDAO) GetMetricsByRunID(runID int) ([]MetricRow, error) {
	rows, err := d.db.Query(`
		SELECT key, x_value, y_value, logged_at
		FROM metrics
		WHERE run_id = $1
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
func (d *PostgresDAO) UpsertArtifact(runID int, path, uri, artifactType string) error {
	_, err := d.db.Exec(
		`INSERT INTO artifacts (run_id, path, uri, type)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (run_id, path) DO UPDATE
		 SET uri = EXCLUDED.uri, type = EXCLUDED.type`,
		runID, path, uri, artifactType,
	)
	return err
}

// GetArtifactsByRunID retrieves all artifacts for a run
func (d *PostgresDAO) GetArtifactsByRunID(runID int) ([]ArtifactRow, error) {
	rows, err := d.db.Query(`
		SELECT path, uri, type
		FROM artifacts
		WHERE run_id = $1
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
func (d *PostgresDAO) GetArtifactByRunIDAndPath(runID int, path string) (*ArtifactRow, error) {
	var a ArtifactRow
	err := d.db.QueryRow(
		"SELECT path, uri, type FROM artifacts WHERE run_id = $1 AND path = $2",
		runID, path,
	).Scan(&a.Path, &a.URI, &a.Type)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateRunNotes updates the notes for a run
func (d *PostgresDAO) UpdateRunNotes(runID int, notes string) error {
	_, err := d.db.Exec(
		"UPDATE runs SET notes = $1 WHERE id = $2",
		notes, runID,
	)
	return err
}
