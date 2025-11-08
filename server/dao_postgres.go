package main

import (
	"database/sql"
	"fmt"
)

// PostgresDAO implements the DAO interface for PostgreSQL
type PostgresDAO struct {
	db *sql.DB
}

// NewPostgresDAO creates a new Postgres DAO
func NewPostgresDAO(db *sql.DB) *PostgresDAO {
	return &PostgresDAO{db: db}
}

// InsertRun inserts a new run
func (d *PostgresDAO) InsertRun(uuid, name string) error {
	_, err := d.db.Exec(
		"INSERT INTO runs (uuid, name) VALUES ($1, $2)",
		uuid, name,
	)
	return err
}

// GetRunByUUID retrieves a run by its UUID
func (d *PostgresDAO) GetRunByUUID(uuid string) (*Run, error) {
	var name string
	err := d.db.QueryRow(
		"SELECT name FROM runs WHERE uuid = $1",
		uuid,
	).Scan(&name)
	if err != nil {
		return nil, err
	}
	return &Run{UUID: uuid, Name: name}, nil
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
func (d *PostgresDAO) InsertMetric(runID int, key string, value float64, loggedAt int64, time *float64, step *int) error {
	_, err := d.db.Exec(
		"INSERT INTO metrics (run_id, key, value, logged_at, time, step) VALUES ($1, $2, $3, to_timestamp($4 / 1000.0), $5, $6)",
		runID, key, value, loggedAt, time, step,
	)
	return err
}

// GetMetricsByRunID retrieves all metrics for a run
func (d *PostgresDAO) GetMetricsByRunID(runID int) ([]MetricRow, error) {
	rows, err := d.db.Query(`
		SELECT key, value, logged_at, time, step
		FROM metrics
		WHERE run_id = $1
		ORDER BY key, step, time
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []MetricRow
	for rows.Next() {
		var m MetricRow
		if err := rows.Scan(&m.Key, &m.Value, &m.LoggedAt, &m.Time, &m.Step); err != nil {
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
