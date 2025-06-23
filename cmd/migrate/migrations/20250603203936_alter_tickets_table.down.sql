-- Remove status column
ALTER TABLE tickets
  DROP COLUMN status;

ALTER TABLE tickets
  ALTER COLUMN guest_id SET NOT NULL;

