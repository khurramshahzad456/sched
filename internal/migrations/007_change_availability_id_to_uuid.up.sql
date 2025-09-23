-- Change availability_rules id column from SERIAL to UUID
-- First, add a new UUID column
ALTER TABLE availability_rules ADD COLUMN new_id UUID DEFAULT gen_random_uuid();

-- Update existing records with new UUIDs
UPDATE availability_rules SET new_id = gen_random_uuid();

-- Drop the old primary key constraint
ALTER TABLE availability_rules DROP CONSTRAINT availability_rules_pkey;

-- Drop the old id column
ALTER TABLE availability_rules DROP COLUMN id;

-- Rename new_id to id
ALTER TABLE availability_rules RENAME COLUMN new_id TO id;

-- Add primary key constraint on the new UUID column
ALTER TABLE availability_rules ADD PRIMARY KEY (id);
