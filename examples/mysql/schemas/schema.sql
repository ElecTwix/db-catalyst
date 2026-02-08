-- MySQL Example Schema for db-catalyst
-- Note: Uses SQLite-compatible syntax that MySQL also supports
-- Some MySQL-specific features like ENUM, JSON, and FULLTEXT are not yet supported by the parser

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at TEXT
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    status TEXT DEFAULT 'draft',
    view_count INTEGER DEFAULT 0,
    created_at TEXT
);

CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TEXT
);

CREATE TABLE post_tags (
    post_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    created_at TEXT,
    PRIMARY KEY (post_id, tag_id)
);
