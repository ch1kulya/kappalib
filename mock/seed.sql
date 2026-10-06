INSERT INTO public.announcements
    VALUES (1, 'Lorem ipsum dolor.', 'Lorem ipsum dolor sit amet consectetur adipiscing elit. Sit amet consectetur adipiscing elit quisque faucibus ex. Adipiscing elit quisque faucibus ex sapien vitae pellentesque.', 'Lorem.', 'https://picsum.photos/1280/720', TRUE, '2026-08-24 14:18:35.701887+00')
ON CONFLICT
    DO NOTHING;

INSERT INTO public.announcements
    VALUES (2, 'Lorem ipsum dolor sit.', 'Lorem ipsum dolor sit amet consectetur adipiscing elit. Amet consectetur adipiscing elit quisque faucibus ex sapien. Quisque faucibus ex sapien vitae pellentesque sem placerat. Vitae pellentesque sem placerat in id cursus mi.', 'Lorem ipsum.', 'https://placebear.com/1280/720', TRUE, '2026-08-24 14:19:31.691969+00')
ON CONFLICT
    DO NOTHING;

INSERT INTO public.novels
    VALUES ('nvl_t923mfvz', 'Lorem ipsum dolor.', 'Lorem ipsum dolor.', 'Lorem.', 2010, 2015, 'ongoing', 'Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.', '18+', 'https://picsum.photos/300/450', '2026-08-24 14:22:07.416794+00', DEFAULT, DEFAULT, DEFAULT, 1, NULL, NULL, 0, 1, '{}', TRUE, TRUE, TRUE, TRUE, '["Lorem ipsum dolor sit amet consectetur adipiscing elit.", "Dolor sit amet consectetur adipiscing elit quisque faucibus."]', 'loremipsumdolorsitametconsecteturadipiscingelitdolorsitametconsecteturadipiscingelitquisquefaucibus', TRUE)
ON CONFLICT
    DO NOTHING;

INSERT INTO public.sources
    VALUES (1, 'Lorem ipsum.', 'https://picsum.photos/100/100', 1, 1, 4476, 'Источник перевода')
ON CONFLICT
    DO NOTHING;

INSERT INTO public.chapters
    VALUES ('chp_y1jinlwz', 'nvl_t923mfvz', 1, 'Vitae pellentesque sem placerat in id cursus mi.', 'Quisque faucibus ex sapien vitae pellentesque sem placerat.', '<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>

<p>Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien vitae pellentesque sem placerat. In id cursus mi pretium tellus duis convallis. Tempus leo eu aenean sed diam urna tempor. Pulvinar vivamus fringilla lacus nec metus bibendum egestas. Iaculis massa nisl malesuada lacinia integer nunc posuere. Ut hendrerit semper vel class aptent taciti sociosqu. Ad litora torquent per conubia nostra inceptos himenaeos.</p>', '2026-08-24 14:23:57.360849+00', 1)
ON CONFLICT
    DO NOTHING;

SELECT
    pg_catalog.setval('public.sources_id_seq', max(id))
FROM
    public.sources;

SELECT
    pg_catalog.setval('public.tags_id_seq', max(id))
FROM
    public.tags;

CREATE TEMP TABLE mock_sources AS
SELECT
    position,
    name
FROM
    unnest(ARRAY['Лунный свиток', 'Бамбуковый павильон', 'Северный перевод', 'Тихая гавань', 'Шёпот страниц'])
    WITH ORDINALITY AS s (name, position);

INSERT INTO public.sources (name, logo_url)
SELECT
    name,
    'https://picsum.photos/seed/kappalib-source-' || position || '/100/100'
FROM
    mock_sources
ON CONFLICT (name)
    DO NOTHING;

CREATE TEMP TABLE mock_tags AS
SELECT
    position,
    name
FROM
    unnest(ARRAY['Боевик', 'Боевые искусства', 'Гарем', 'Детектив', 'Драма', 'Исекай', 'Историческое', 'Комедия', 'Культивация', 'Магия', 'Мистика', 'Научная фантастика', 'Повседневность', 'Постапокалипсис', 'Приключения', 'Психология', 'Реинкарнация', 'Романтика', 'Система', 'Сёнэн', 'Трагедия', 'Ужасы', 'Фэнтези', 'Школа'])
    WITH ORDINALITY AS t (name, position);

INSERT INTO public.tags (name)
SELECT
    name
FROM
    mock_tags
ON CONFLICT (name)
    DO NOTHING;

CREATE TEMP TABLE mock_novels AS
SELECT
    i,
    'nvl_mock' || lpad(i::text, 4, '0') AS id,
    (ARRAY['Тень', 'Легенда', 'Хроники', 'Путь', 'Возвращение', 'Наследие', 'Сердце', 'Песнь', 'Врата', 'Империя', 'Академия', 'Клинок'])[(i - 1) % 12 + 1] || ' ' || (ARRAY['северного ветра', 'забытого бога', 'алого дракона', 'последнего мага', 'девяти небес', 'лунной башни', 'тысячи миров', 'пепельного короля', 'звёздного моря', 'нефритового императора'])[((i - 1) % 12 * 3 + (i - 1) / 12 * 2) % 10 + 1] AS title,
    'The ' || (ARRAY['Shadow', 'Legend', 'Chronicles', 'Path', 'Return', 'Legacy', 'Heart', 'Song', 'Gates', 'Empire', 'Academy', 'Blade'])[(i - 1) % 12 + 1] || ' ' || (ARRAY['of the North Wind', 'of the Forgotten God', 'of the Crimson Dragon', 'of the Last Mage', 'of the Nine Heavens', 'of the Lunar Tower', 'of a Thousand Worlds', 'of the Ashen King', 'of the Starry Sea', 'of the Jade Emperor'])[((i - 1) % 12 * 3 + (i - 1) / 12 * 2) % 10 + 1] AS title_en,
    (ARRAY['Тихий Лис', 'Бумажный Журавль', 'Мэй Лин', 'Сон Ынха', 'Хару Аояма', 'Чжоу Юнь', 'Ветер Востока', 'Ли Сяо', 'Акира Мидзуно', 'Фэн Цин', 'Пак Джиын', 'Серебряная Ива'])[(i * 5) % 12 + 1] AS author,
    2005 + (i * 7) % 20 AS year_start,
    (ARRAY['ongoing', 'ongoing', 'ongoing', 'completed', 'completed', 'announced'])[i % 6 + 1] AS status,
    (ARRAY['Проснувшись в чужом теле накануне казни, бывший стратег империи получает второй шанс. Теперь ему предстоит распутать заговор, погубивший его в прошлой жизни, и не повторить старых ошибок.', 'Небольшая горная школа, забытая всеми сектами, внезапно принимает ученика, чья духовная сила не поддаётся измерению. Старейшины не знают, радоваться им или готовиться к беде.', 'В мире, где каждый получает системный класс в шестнадцать лет, героине достаётся роль, о которой никто прежде не слышал. Чтобы выжить, ей придётся самой разобраться в правилах этой игры.', 'Тихая жизнь владельца книжной лавки заканчивается в тот день, когда на пороге появляется раненый наследник враждующего клана с рукописью, за которую готовы убивать.', 'Экспедиция к последнему маяку на краю известного мира должна была занять неделю. Прошёл год, и из всей команды назад вернулся лишь картограф — с картами земель, которых не существует.', 'Студентка магической академии случайно заключает контракт с древним духом, запечатанным в библиотеке. Дух обещает силу, но у каждой его услуги есть цена.'])[(i * 7) % 6 + 1] AS description,
    (ARRAY['12+', '16+', '18+', '16+', '12+', '6+'])[i % 6 + 1] AS age_rating,
    CASE WHEN i % 6 = 5 THEN
        0
    WHEN i % 10 = 7 THEN
        300 + (i * 13) % 200
    ELSE
        5 + (i * 37) % 160
    END AS chapters_total,
    now() - i * 9 * interval '1 hour' AS last_update
FROM
    generate_series(1, 60) AS i;

INSERT INTO public.novels (id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at, views_count, has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex, has_profanity)
SELECT
    id,
    title,
    title_en,
    author,
    year_start,
    CASE WHEN status = 'completed' THEN
        LEAST (year_start + 1 + i % 3, 2026)
    END,
    status,
    description,
    age_rating,
    'https://picsum.photos/seed/kappalib-' || id || '/300/450',
    last_update - (chapters_total / 10 * 2 + 1) * interval '1 day',
    (i * 7919) % 48000 + (60 - i) * 150,
    i % 11 = 0,
    i % 13 = 0,
    i % 17 = 0,
    i % 12 = 2,
    i % 4 = 0
FROM
    mock_novels
ON CONFLICT
    DO NOTHING;

INSERT INTO public.novel_tags (novel_id, tag_id)
SELECT
    n.id,
    t.id
FROM
    mock_novels AS n
    CROSS JOIN LATERAL generate_series(1, 2 + n.i % 4) AS s
    INNER JOIN mock_tags AS mt ON mt.position = (n.i * 7 + s * 5) % 24 + 1
    INNER JOIN public.tags AS t ON t.name = mt.name
ON CONFLICT
    DO NOTHING;

INSERT INTO public.novel_tags (novel_id, tag_id)
SELECT
    'nvl_t923mfvz',
    id
FROM
    public.tags
WHERE
    name IN ('Приключения', 'Фэнтези')
ON CONFLICT
    DO NOTHING;

WITH texts AS (
    SELECT
        ARRAY['Ветер гнал по небу рваные облака, и в их просветах то и дело вспыхивали холодные звёзды. Лин Юэ стоял у края обрыва и смотрел на огни далёкого города, пытаясь понять, в какой момент всё пошло не так. Ещё утром у него был дом, учитель и ясное будущее.',
        'Старик неторопливо разлил чай по пиалам и лишь затем поднял взгляд. В его глазах не было ни удивления, ни страха — только усталость человека, который слишком долго ждал этого разговора. «Садись, — сказал он. — Нам нужно многое обсудить, а времени осталось мало».',
        'Коридоры академии в этот час пустели, и шаги отдавались гулким эхом под сводчатым потолком. Мэй прижимала к груди тяжёлый фолиант и старалась не думать о том, что будет, если её заметит кто-нибудь из наставников. Библиотека закрылась ещё два часа назад.',
        'Клинок вышел из ножен беззвучно, будто сам воздух расступился перед ним. Противник замер, и на мгновение над площадью повисла такая тишина, что стало слышно, как потрескивают факелы. Никто из собравшихся не решался даже вдохнуть.',
        'Письмо было коротким: всего три строки, написанные торопливым, неровным почерком. Но чем дольше она вглядывалась в них, тем яснее понимала, что прежняя жизнь закончилась. Где-то за стеной часы пробили полночь, и этот звук показался ей похожим на приговор.',
        'Караван остановился у высохшего колодца, когда солнце уже клонилось к закату. Проводник долго молчал, разглядывая следы на песке, а потом покачал головой и велел разбивать лагерь. До ближайшего оазиса оставалось два дня пути, и никто не хотел идти туда ночью.'] AS paragraphs,
        ARRAY['Начало пути',
        'Незваный гость',
        'Тени прошлого',
        'Первое испытание',
        'Сделка',
        'Ночь перед бурей',
        'Старые долги',
        'Шёпот в темноте',
        'Цена обещания',
        'Возвращение домой',
        'Разбитое зеркало',
        'Последний рубеж',
        'Без названия',
        'Письмо без подписи',
        'Рассвет'] AS titles
)
INSERT INTO public.chapters (id, novel_id, chapter_num, title, title_en, content, source_id, created_at)
SELECT
    'chp_m' || lpad(n.i::text, 3, '0') || lpad(c::text, 4, '0'),
    n.id,
    c,
    t.titles[(c + n.i) % 15 + 1],
    'Chapter ' || c,
    repeat('<p>' || t.paragraphs[(c + n.i) % 6 + 1] || '</p>' || E'\n\n' || '<p>' || t.paragraphs[(c * 2 + n.i + 1) % 6 + 1] || '</p>' || E'\n\n', 4 + (c * 7 + n.i) % 9),
    src.id,
    n.last_update - (n.chapters_total - c) / 10 * interval '2 days' - (n.chapters_total - c) % 10 * interval '1 minute'
FROM
    mock_novels AS n
    CROSS JOIN texts AS t
    CROSS JOIN LATERAL generate_series(1, n.chapters_total) AS c
    INNER JOIN mock_sources AS ms ON ms.position = CASE WHEN n.i % 3 = 0
        AND c > n.chapters_total / 2 THEN
        (n.i + 1) % 5 + 1
    ELSE
        n.i % 5 + 1
    END
    INNER JOIN public.sources AS src ON src.name = ms.name
ON CONFLICT
    DO NOTHING;

DROP TABLE mock_novels, mock_tags, mock_sources;

SELECT
    pg_catalog.setval('public.announcements_id_seq', 2, TRUE);

