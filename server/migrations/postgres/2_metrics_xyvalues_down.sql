ALTER TABLE metrics ADD COLUMN time DOUBLE PRECISION;
ALTER TABLE metrics ADD COLUMN step INTEGER;

UPDATE metrics
    SET time = x_value;

ALTER TABLE metrics DROP COLUMN x_value;
ALTER TABLE metrics RENAME COLUMN y_value TO value;
    
