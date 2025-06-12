-- 1. Add qr_code_urls to guests
ALTER TABLE guests
ADD COLUMN qr_code_urls TEXT[];

-- 2. Drop qr_code_urls from tickets
ALTER TABLE tickets
DROP COLUMN IF EXISTS qr_codes_urls;
