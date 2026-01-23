-- Add parent_run_id and nesting_level columns to runs table
ALTER TABLE runs ADD COLUMN parent_run_id INTEGER;
ALTER TABLE runs ADD COLUMN nesting_level INTEGER DEFAULT 0;

-- Create indexes for efficient queries
CREATE INDEX idx_runs_parent_run_id ON runs(parent_run_id);
CREATE INDEX idx_runs_nesting_level ON runs(nesting_level);
