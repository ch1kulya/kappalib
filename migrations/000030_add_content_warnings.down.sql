UPDATE
    novels
SET
    age_rating = '19+'
WHERE
    has_self_harm
    OR has_drug_usage
    OR has_sexual_violence
    OR has_graphic_sex;

DROP INDEX IF EXISTS idx_novels_content_warnings;

ALTER TABLE novels
    DROP COLUMN has_self_harm,
    DROP COLUMN has_drug_usage,
    DROP COLUMN has_sexual_violence,
    DROP COLUMN has_graphic_sex;

