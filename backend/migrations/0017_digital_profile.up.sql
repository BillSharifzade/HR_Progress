-- 0017: digital employee profile — the structured half of the questionnaire
-- ("Анкета формирования цифрового профиля сотрудника", 146 responses).
--
-- Four new stores plus an extension to certifications. The split follows what
-- the answers actually look like, measured against the source file rather than
-- guessed:
--
--   * columns with a fixed option set (education, experience band, career goal,
--     the four readiness questions, learning hours) and the multi-selects that
--     have a canonical option list with a small free-text tail (development
--     directions, professional interests, learning formats) become typed
--     columns on `user_profile`;
--   * the six language columns become rows in `user_languages`;
--   * the "which companies did you work for" free text becomes rows in
--     `user_work_experience`;
--   * the genuinely open-ended answers (notable projects, areas of expertise,
--     what colleagues ask about, training topics, free-form note) become rows
--     in `user_survey_answers`, which is a generic question→answer store so a
--     future form round needs no schema change.
--
-- Everything here is operator-editable: every table carries its own PK and an
-- `updated_at` trigger, and nothing is keyed on the import so HR can add rows
-- for people who never filled the form in.

-- ---------------------------------------------------------------- profile ---
CREATE TABLE user_profile (
    user_id                     uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- education: multi-select, e.g. "Высшее, Магистратура"
    education_levels            text[]      NOT NULL DEFAULT '{}',
    institution                 text,
    specialty                   text,

    prior_experience_band       text,
    career_goal                 text,
    development_directions      text[]      NOT NULL DEFAULT '{}',

    -- readiness answers; free text rather than enums because the form's option
    -- wording is owned by HR and has already changed once between rounds
    mobility_readiness          text,
    relocation_readiness        text,
    internal_projects_readiness text,
    teaching_readiness          text,

    professional_interests      text[]      NOT NULL DEFAULT '{}',
    learning_formats            text[]      NOT NULL DEFAULT '{}',
    learning_hours_band         text,

    submitted_at                timestamptz,
    source                      text        NOT NULL DEFAULT 'manual',
    created_at                  timestamptz NOT NULL DEFAULT now(),
    updated_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER user_profile_set_updated_at
    BEFORE UPDATE ON user_profile
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------------------- languages ---
-- One row per language a person claims. The form rendered its CEFR grid as
-- checkboxes, so 27 people ticked several levels for a single language and 5
-- ticked all six; the importer keeps the highest. Note the source data mixes
-- alphabets — "А1"/"А2" arrive with a CYRILLIC А while "B1".."C2" are Latin —
-- so values are normalized to Latin before they reach this CHECK.
CREATE TABLE user_languages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language   text        NOT NULL,
    level      text        NOT NULL CHECK (level IN ('A1','A2','B1','B2','C1','C2')),
    source     text        NOT NULL DEFAULT 'manual',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_languages_lang_not_blank CHECK (btrim(language) <> '')
);

CREATE UNIQUE INDEX user_languages_uniq ON user_languages (user_id, lower(language));
CREATE INDEX user_languages_user_idx ON user_languages (user_id);

CREATE TRIGGER user_languages_set_updated_at
    BEFORE UPDATE ON user_languages
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------------- work experience ---
-- Employment before ЗАО "КОИНОТИ НАВ". Deliberately separate from
-- `user_history`, which records movements INSIDE the company (rotations,
-- promotions) and is keyed on an event date this data does not have.
--
-- `raw_text` keeps the fragment the importer split this row out of, so a bad
-- split stays traceable and correctable instead of silently becoming truth.
CREATE TABLE user_work_experience (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company     text        NOT NULL,
    position    text,
    started_on  date,
    ended_on    date,
    description text,
    sort_order  int         NOT NULL DEFAULT 0,
    raw_text    text,
    source      text        NOT NULL DEFAULT 'manual',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_work_experience_company_not_blank CHECK (btrim(company) <> ''),
    CONSTRAINT user_work_experience_dates CHECK (
        started_on IS NULL OR ended_on IS NULL OR ended_on >= started_on
    )
);

CREATE INDEX user_work_experience_user_idx
    ON user_work_experience (user_id, sort_order, created_at);

CREATE TRIGGER user_work_experience_set_updated_at
    BEFORE UPDATE ON user_work_experience
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --------------------------------------------------------- survey answers ---
-- Generic question→answer store for the open-ended columns. `question_code` is
-- a stable slug so a re-import updates an answer in place, while
-- `question_text` keeps the exact wording the person actually read — the form
-- is HR-owned and its phrasing drifts between rounds.
CREATE TABLE user_survey_answers (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_key      text        NOT NULL DEFAULT 'digital_profile',
    question_code text        NOT NULL,
    question_text text        NOT NULL,
    answer_text   text        NOT NULL,
    position      int         NOT NULL DEFAULT 0,
    submitted_at  timestamptz,
    source        text        NOT NULL DEFAULT 'manual',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_survey_answers_answer_not_blank CHECK (btrim(answer_text) <> '')
);

CREATE UNIQUE INDEX user_survey_answers_uniq
    ON user_survey_answers (user_id, form_key, question_code);
CREATE INDEX user_survey_answers_user_idx
    ON user_survey_answers (user_id, position);

CREATE TRIGGER user_survey_answers_set_updated_at
    BEFORE UPDATE ON user_survey_answers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------- certifications: files ---
-- Certificates arrive two ways and both must work: as a link (the 24 responses
-- in this round are all Google Drive URLs) or as an uploaded document. Exactly
-- one of the two is present per row.
ALTER TABLE user_certifications
    ADD COLUMN source_url    text,
    ADD COLUMN file_path     text,
    ADD COLUMN file_name     text,
    ADD COLUMN file_size     bigint,
    ADD COLUMN content_type  text,
    ADD COLUMN source        text NOT NULL DEFAULT 'manual';

-- A file is stored as a path relative to the upload root plus its metadata;
-- either all of that is present or none of it is. `title` alone remains valid
-- (a certificate with neither a scan nor a link is still a fact worth holding).
ALTER TABLE user_certifications
    ADD CONSTRAINT user_certifications_file_complete CHECK (
        (file_path IS NULL AND file_name IS NULL AND file_size IS NULL AND content_type IS NULL)
     OR (file_path IS NOT NULL AND file_name IS NOT NULL AND file_size IS NOT NULL)
    );

ALTER TABLE user_certifications
    ADD CONSTRAINT user_certifications_not_both CHECK (
        source_url IS NULL OR file_path IS NULL
    );
