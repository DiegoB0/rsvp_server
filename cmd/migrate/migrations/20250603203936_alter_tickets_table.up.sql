-- Make guest_id nullable
ALTER TABLE tickets
  ALTER COLUMN guest_id DROP NOT NULL;

-- Add status column with CHECK constraint
ALTER TABLE tickets
  ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'used'));

