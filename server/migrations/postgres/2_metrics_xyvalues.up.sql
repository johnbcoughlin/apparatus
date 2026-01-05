ALTER TABLE metrics ADD COLUMN x_value DOUBLE PRECISION;
ALTER TABLE metrics RENAME COLUMN value TO y_value;

UPDATE metrics
    SET x_value = COALESCE(time, step);

ALTER TABLE metrics DROP COLUMN time;
ALTER TABLE metrics DROP COLUMN step;
