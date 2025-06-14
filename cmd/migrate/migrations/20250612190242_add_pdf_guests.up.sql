-- 1. Add pdf files to guests
ALTER TABLE guests
ADD COLUMN pdf_files TEXT[];
