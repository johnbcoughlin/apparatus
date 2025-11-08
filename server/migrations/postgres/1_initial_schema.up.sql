CREATE TABLE IF NOT EXISTS runs (
    id SERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS parameters (
    id SERIAL PRIMARY KEY,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value_type TEXT NOT NULL,
    value_string TEXT,
    value_bool BOOLEAN,
    value_float DOUBLE PRECISION,
    value_int INTEGER,
    UNIQUE(run_id, key)
);

CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    run_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    logged_at TIMESTAMP NOT NULL,
    time DOUBLE PRECISION,
    step INTEGER,
    UNIQUE(run_id, key, time, step)
);

CREATE TABLE IF NOT EXISTS artifacts (
    id SERIAL PRIMARY KEY,
    run_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    uri TEXT NOT NULL,
    type TEXT NOT NULL,
    UNIQUE(run_id, path)
);
