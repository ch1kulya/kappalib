ALTER TABLE novels
ADD COLUMN has_self_harm BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN has_drug_usage BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN has_sexual_violence BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN has_graphic_sex BOOLEAN NOT NULL DEFAULT false;

UPDATE novels
SET
    age_rating = '18+',
    has_self_harm = true,
    has_drug_usage = true,
    has_sexual_violence = true,
    has_graphic_sex = true
WHERE age_rating = '19+';

CREATE INDEX IF NOT EXISTS idx_novels_content_warnings
ON novels (has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex)
WHERE has_self_harm OR has_drug_usage OR has_sexual_violence OR has_graphic_sex;
