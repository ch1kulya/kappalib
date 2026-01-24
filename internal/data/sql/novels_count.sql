SELECT COUNT(*) FROM novels
WHERE NOT (has_self_harm OR has_drug_usage OR has_sexual_violence OR has_graphic_sex);
