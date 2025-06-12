-- 1. Re-add qr_code_urls to tickets
ALTER TABLE tickets
ADD COLUMN qr_codes_urls TEXT[];

-- 2. Drop qr_code_urls from guests
ALTER TABLE guests
DROP COLUMN IF EXISTS qr_code_urls;

