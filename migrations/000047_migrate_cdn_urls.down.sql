UPDATE
    novels
SET
    cover_url = REPLACE(cover_url, 'https://cdn.kappalib.rip', 'https://s3.kappalib.rip')
WHERE
    cover_url LIKE '%https://cdn.kappalib.rip%';

UPDATE
    sources
SET
    logo_url = REPLACE(logo_url, 'https://cdn.kappalib.rip', 'https://s3.kappalib.rip')
WHERE
    logo_url LIKE '%https://cdn.kappalib.rip%';

UPDATE
    chapters
SET
    content = REPLACE(content, 'https://cdn.kappalib.rip', 'https://s3.kappalib.rip')
WHERE
    content LIKE '%https://cdn.kappalib.rip%';

UPDATE
    users
SET
    cookies = REPLACE(cookies::text, 'https://cdn.kappalib.rip', 'https://s3.kappalib.rip')::jsonb
WHERE
    cookies::text LIKE '%https://cdn.kappalib.rip%';

