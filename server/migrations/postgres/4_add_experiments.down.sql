-- Drop index
DROP INDEX IF EXISTS idx_runs_experiment_id;

-- Remove experiment_id from runs
ALTER TABLE runs DROP COLUMN experiment_id;

-- Drop experiments table
DROP TABLE IF EXISTS experiments;
