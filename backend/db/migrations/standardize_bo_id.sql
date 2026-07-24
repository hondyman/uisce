-- Standardize User-Facing Business Object IDs to 'business_object_id'
-- This ensures consistency across the platform (API, DB, Frontend, Backend)

-- 1. Rename column in bo_fields
ALTER TABLE bo_fields RENAME COLUMN bo_id TO business_object_id;

-- 2. Rename column in page_layouts
ALTER TABLE page_layouts RENAME COLUMN bo_id TO business_object_id;

-- 3. Rename column in form_submissions
ALTER TABLE form_submissions RENAME COLUMN bo_id TO business_object_id;

-- 4. Rename column in semantic.bo_sql_cache
ALTER TABLE semantic.bo_sql_cache RENAME COLUMN bo_id TO business_object_id;

-- 5. Rename constraints (optional but good for consistency)
-- Note: DB might have auto-generated names, but we attempt standard ones if they matched our schema file
ALTER TABLE bo_fields RENAME CONSTRAINT fk_field_bo TO fk_field_business_object;
ALTER TABLE bo_fields RENAME CONSTRAINT unique_field_per_bo TO unique_field_per_business_object;

ALTER TABLE page_layouts RENAME CONSTRAINT fk_layout_bo TO fk_layout_business_object;
ALTER TABLE page_layouts RENAME CONSTRAINT unique_layout_name TO unique_layout_name_per_business_object;

ALTER TABLE form_submissions RENAME CONSTRAINT fk_submission_bo TO fk_submission_business_object;
