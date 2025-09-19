-- Remove timezone column from availability_rules table
-- All times are now in UTC
ALTER TABLE availability_rules DROP COLUMN IF EXISTS timezone;
