ALTER TABLE image_generation_jobs
    ADD COLUMN IF NOT EXISTS reference_image_file_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reference_image_original_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reference_image_mime_type VARCHAR(100) NOT NULL DEFAULT '';

