-- Users table catalog entry
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    profile_id INTEGER REFERENCES profiles(id),
    CONSTRAINT users_email_unique UNIQUE (email)
) WITHOUT ROWID;

CREATE UNIQUE INDEX idx_users_email ON users (email);

-- Profiles catalog entry
CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    bio TEXT DEFAULT 'none'
);

-- Active users view
CREATE VIEW active_users AS
SELECT u.id, u.email
FROM users u
WHERE u.profile_id IS NOT NULL;
