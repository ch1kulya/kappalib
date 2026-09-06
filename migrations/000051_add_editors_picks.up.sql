CREATE TABLE editors_picks (
    novel_id varchar(20) PRIMARY KEY REFERENCES novels(id) ON DELETE CASCADE,
    position integer NOT NULL DEFAULT 0
);
