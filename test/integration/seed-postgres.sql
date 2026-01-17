-- PostgreSQL seed data
-- More comprehensive test data

-- Add more users
INSERT INTO users (username, email, password_hash, role) VALUES
    ('david', 'david@example.com', 'hashed_password', 'user'),
    ('eve', 'eve@example.com', 'hashed_password', 'user'),
    ('frank', 'frank@example.com', 'hashed_password', 'admin'),
    ('grace', 'grace@example.com', 'hashed_password', 'user')
ON CONFLICT (username) DO UPDATE SET updated_at = NOW();

-- Add more posts
INSERT INTO posts (user_id, title, slug, content, excerpt, status, views_count, published_at) VALUES
    (1, 'Advanced Go Patterns', 'advanced-go-patterns', 'Deep dive into Go patterns...', 'Learn advanced patterns', 'published', 150, NOW()),
    (2, 'Database Best Practices', 'db-best-practices', 'Database optimization tips...', 'Optimize your database', 'published', 200, NOW()),
    (3, 'SQL Injection Prevention', 'sql-injection', 'How to prevent SQL injection...', 'Stay safe', 'published', 500, NOW()),
    (1, 'Rust vs Go', 'rust-vs-go', 'Comparing Rust and Go...', 'The debate', 'published', 300, NOW()),
    (2, 'Testing in Go', 'testing-go', 'Unit testing and integration testing...', 'Test all the things', 'published', 100, NOW())
ON CONFLICT (slug) DO UPDATE SET updated_at = NOW();

-- Add tags
INSERT INTO tags (name, slug, description, post_count) VALUES
    ('programming', 'programming', 'Programming related', 5),
    ('security', 'security', 'Security topics', 2),
    ('comparison', 'comparison', 'Comparison articles', 1)
ON CONFLICT (slug) DO UPDATE SET post_count = tags.post_count + 1;

-- Link posts to tags (with proper foreign key references)
INSERT INTO post_tags (post_id, tag_id) VALUES
    (1, 1),
    (2, 1),
    (3, 2),
    (4, 3),
    (5, 1)
ON CONFLICT DO NOTHING;

-- Add comments
INSERT INTO comments (post_id, user_id, parent_id, content, is_approved) VALUES
    (1, 2, NULL, 'Great article!', TRUE),
    (1, 3, 1, 'Thanks for reading!', TRUE),
    (2, 1, NULL, 'Very helpful tips.', TRUE),
    (3, 4, NULL, 'This saved my project!', TRUE),
    (4, 2, NULL, 'I prefer Go personally.', TRUE)
ON CONFLICT DO NOTHING;

-- Update post counts
UPDATE tags t
SET post_count = (SELECT COUNT(*) FROM post_tags pt WHERE pt.tag_id = t.id);
