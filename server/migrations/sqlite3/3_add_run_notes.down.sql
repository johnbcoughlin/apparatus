-- SQLite doesn't support DROP COLUMN, so we recreate the table
CREATE TABLE runs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO runs_new (id, uuid, name, created_at)
SELECT id, uuid, name, created_at FROM runs;

DROP TABLE runs;

ALTER TABLE runs_new RENAME TO runs;
