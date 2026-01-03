CREATE TABLE items (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    metadata JSON
) WITHOUT ROWID;

CREATE TABLE tags (
    tag TEXT PRIMARY KEY,
    description TEXT
) STRICT;

CREATE TABLE item_tags (
    item_id INTEGER REFERENCES items(id),
    tag TEXT REFERENCES tags(tag),
    PRIMARY KEY (item_id, tag)
);
