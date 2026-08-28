ALTER TABLE novels
    ADD COLUMN has_self_harm boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN has_drug_usage boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN has_sexual_violence boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN has_graphic_sex boolean NOT NULL DEFAULT FALSE;

UPDATE
    novels
SET
    age_rating = '18+',
    has_self_harm = TRUE,
    has_drug_usage = TRUE,
    has_sexual_violence = TRUE,
    has_graphic_sex = TRUE
WHERE
    age_rating = '19+';

CREATE INDEX IF NOT EXISTS idx_novels_content_warnings ON novels (has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex)
WHERE
    has_self_harm OR has_drug_usage OR has_sexual_violence OR has_graphic_sex;

