-- Remove indexes
DROP INDEX IF EXISTS idx_runs_parent_run_id;
DROP INDEX IF EXISTS idx_runs_nesting_level;

-- Remove columns
ALTER TABLE runs DROP COLUMN parent_run_id;
ALTER TABLE runs DROP COLUMN nesting_level;
