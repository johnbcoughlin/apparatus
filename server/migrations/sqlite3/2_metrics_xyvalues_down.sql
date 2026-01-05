-- SQLite doesn't support DROP COLUMN well with constraints, so we recreate the table
CREATE TABLE metrics_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value REAL NOT NULL,
    logged_at TIMESTAMP NOT NULL,
    time REAL,
    step INTEGER,
    UNIQUE(run_id, key, time, step)
);

INSERT INTO metrics_new (id, run_id, key, value, logged_at, time, step)
SELECT id, run_id, key, y_value, logged_at, x_value, NULL
FROM metrics;

DROP TABLE metrics;

ALTER TABLE metrics_new RENAME TO metrics;
    
