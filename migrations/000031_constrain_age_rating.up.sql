ALTER TABLE novels
ADD CONSTRAINT novels_age_rating_check
CHECK (age_rating IS NULL OR age_rating IN ('0+', '6+', '12+', '16+', '18+'));
