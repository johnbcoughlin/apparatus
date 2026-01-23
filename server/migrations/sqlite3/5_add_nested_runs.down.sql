-- Remove indexes
DROP INDEX IF EXISTS idx_runs_parent_run_id;
DROP INDEX IF EXISTS idx_runs_nesting_level;

-- Remove columns (SQLite 3.35.0+ supports DROP COLUMN)
ALTER TABLE runs DROP COLUMN parent_run_id;
ALTER TABLE runs DROP COLUMN nesting_level;
