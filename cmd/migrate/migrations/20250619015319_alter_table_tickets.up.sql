ALTER TABLE guests
DROP COLUMN pdf_files;

ALTER TABLE guests
ADD COLUMN pdf_files VARCHAR(250);

