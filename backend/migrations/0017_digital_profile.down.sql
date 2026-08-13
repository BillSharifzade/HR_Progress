-- Reverse 0017. Drops the four digital-profile stores and the certification
-- link/file columns.
--
-- This is genuinely destructive: everything imported from the questionnaire
-- and everything HR has since edited by hand lives in these tables, and none
-- of it is reconstructible from the rest of the schema. Take a dump first.

ALTER TABLE user_certifications
    DROP CONSTRAINT IF EXISTS user_certifications_not_both,
    DROP CONSTRAINT IF EXISTS user_certifications_file_complete;

ALTER TABLE user_certifications
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS content_type,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS file_name,
    DROP COLUMN IF EXISTS file_path,
    DROP COLUMN IF EXISTS source_url;

DROP TABLE IF EXISTS user_survey_answers;
DROP TABLE IF EXISTS user_work_experience;
DROP TABLE IF EXISTS user_languages;
DROP TABLE IF EXISTS user_profile;
