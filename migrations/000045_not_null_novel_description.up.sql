UPDATE novels SET description = 'Нет описания.' WHERE description IS NULL;

ALTER TABLE novels ALTER COLUMN description SET DEFAULT 'Нет описания.';
ALTER TABLE novels ALTER COLUMN description SET NOT NULL;
