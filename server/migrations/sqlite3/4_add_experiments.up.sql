-- Create experiments table
CREATE TABLE IF NOT EXISTS experiments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add experiment_id to runs table
ALTER TABLE runs ADD COLUMN experiment_id INTEGER;

-- Create index on experiment_id
CREATE INDEX idx_runs_experiment_id ON runs(experiment_id);

-- Create Default experiment with valid UUID
INSERT INTO experiments (uuid, name, created_at)
VALUES ('00000000-0000-0000-0000-000000000000', 'Default', CURRENT_TIMESTAMP);

-- Migrate all existing runs to Default experiment
UPDATE runs SET experiment_id = (SELECT id FROM experiments WHERE uuid = '00000000-0000-0000-0000-000000000000');
