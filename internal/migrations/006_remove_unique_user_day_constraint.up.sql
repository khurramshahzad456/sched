-- Remove unique constraint to allow multiple availability rules per day
-- Users can now have overlapping availability periods on the same day
ALTER TABLE availability_rules DROP CONSTRAINT IF EXISTS uniq_user_day;
