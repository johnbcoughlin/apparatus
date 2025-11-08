CREATE TABLE IF NOT EXISTS runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS parameters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value_type TEXT NOT NULL,
    value_string TEXT,
    value_bool INTEGER,
    value_float REAL,
    value_int INTEGER,
    UNIQUE(run_id, key)
);

CREATE TABLE IF NOT EXISTS metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value REAL NOT NULL,
    logged_at TIMESTAMP NOT NULL,
    time REAL,
    step INTEGER,
    UNIQUE(run_id, key, time, step)
);

CREATE TABLE IF NOT EXISTS artifacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    uri TEXT NOT NULL,
    type TEXT NOT NULL,
    UNIQUE(run_id, path)
);
