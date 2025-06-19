ALTER TABLE guests
DROP COLUMN IF EXISTS pdf_files;

ALTER TABLE guests
ADD COLUMN pdf_files TEXT[];

