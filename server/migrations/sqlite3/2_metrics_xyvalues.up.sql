-- SQLite doesn't support DROP COLUMN well with constraints, so we recreate the table
CREATE TABLE metrics_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    x_value REAL NOT NULL,
    y_value REAL NOT NULL,
    logged_at TIMESTAMP NOT NULL,
    UNIQUE(run_id, key, x_value)
);

INSERT INTO metrics_new (id, run_id, key, x_value, y_value, logged_at)
SELECT id, run_id, key, COALESCE(time, step), value, logged_at
FROM metrics;

DROP TABLE metrics;

ALTER TABLE metrics_new RENAME TO metrics;
