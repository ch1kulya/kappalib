CREATE OR REPLACE FUNCTION update_alt_titles_norm()
RETURNS TRIGGER AS $$
BEGIN
    NEW.alt_titles_norm := lower(regexp_replace(
        COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(NEW.alt_titles)), ''),
        '[^[:alnum:]]', '', 'g'
    ));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

UPDATE novels SET alt_titles = alt_titles WHERE alt_titles != '[]'::JSONB;
