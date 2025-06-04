-- Remove status column
ALTER TABLE tickets
  DROP COLUMN status;

-- Make guest_id NOT NULL again
-- Note: You must ensure no nulls exist before applying this or it will fail
ALTER TABLE tickets
  ALTER COLUMN guest_id SET NOT NULL;

