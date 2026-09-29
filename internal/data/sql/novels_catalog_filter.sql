SELECT
    n.id
FROM
    novels AS n
WHERE (coalesce(cardinality($1::text[]), 0) = 0
    OR n.status = ANY ($1::text[]))
AND ($2::integer IS NULL
    OR n.year_start >= $2)
AND ($3::integer IS NULL
    OR n.year_start <= $3)
AND ($4::integer IS NULL
    OR n.chapters_count >= $4)
AND ($5::integer IS NULL
    OR n.chapters_count <= $5)
AND (coalesce(cardinality($6::integer[]), 0) = 0
    OR n.id IN (
        SELECT
            nt.novel_id
        FROM
            novel_tags AS nt
        WHERE
            nt.tag_id = ANY ($6::integer[])
        GROUP BY
            nt.novel_id
        HAVING
            count(*) = cardinality($6::integer[])))
