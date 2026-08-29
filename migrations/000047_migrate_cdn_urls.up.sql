UPDATE novels
SET cover_url = REPLACE(cover_url, 'https://s3.kappalib.rip', 'https://cdn.kappalib.rip')
WHERE cover_url LIKE '%https://s3.kappalib.rip%';

UPDATE sources
SET logo_url = REPLACE(logo_url, 'https://s3.kappalib.rip', 'https://cdn.kappalib.rip')
WHERE logo_url LIKE '%https://s3.kappalib.rip%';

UPDATE chapters
SET content = REPLACE(content, 'https://s3.kappalib.rip', 'https://cdn.kappalib.rip')
WHERE content LIKE '%https://s3.kappalib.rip%';

UPDATE users
SET cookies = REPLACE(cookies::text, 'https://s3.kappalib.rip', 'https://cdn.kappalib.rip')::jsonb
WHERE cookies::text LIKE '%https://s3.kappalib.rip%';
