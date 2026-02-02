-- PostgreSQL example schema demonstrating various PostgreSQL-specific types

-- Users table with UUID primary key and JSONB metadata
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT,
    useremail TEXT,
    metadata JSONB,
    tags TEXT[],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Posts table with foreign key and array types
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    title TEXT,
    postbody TEXT,
    categories TEXT[],
    view_count INTEGER DEFAULT 0,
    rating NUMERIC,
    is_published BOOLEAN DEFAULT false,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID REFERENCES posts(id),
    user_id UUID REFERENCES users(id),
    commentbody TEXT,
    likes INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tags table for many-to-many relationship
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tagname TEXT,
    tagdescription TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Post tags junction table
CREATE TABLE post_tags (
    post_id UUID REFERENCES posts(id),
    tag_id UUID REFERENCES tags(id),
    PRIMARY KEY (post_id, tag_id)
);

-- Basic indexes
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_comments_post_id ON comments(post_id);
