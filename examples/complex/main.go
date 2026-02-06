//nolint:gocritic // Example code uses log.Fatal after defer for simplicity.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	complexdb "github.com/electwix/db-catalyst/examples/complex/db"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	// Open database
	sqlDB, err := sql.Open("modernc.org/sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()

	// Initialize schema
	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		_ = sqlDB.Close()
		log.Print(err)
		return
	}

	queries := complexdb.New(sqlDB)

	// Create authors
	author1, err := queries.CreateAuthor(ctx, "Alice Smith", "alice@example.com", sql.NullString{String: "Go enthusiast", Valid: true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created author: %s\n", author1.Name)

	author2, err := queries.CreateAuthor(ctx, "Bob Jones", "bob@example.com", sql.NullString{String: "SQLite expert", Valid: true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created author: %s\n", author2.Name)

	// Create tags
	tags := []struct {
		name        string
		description string
	}{
		{"go", "Go programming language"},
		{"sqlite", "SQLite database"},
		{"tutorial", "Tutorial content"},
		{"advanced", "Advanced topics"},
	}

	tagIDs := make(map[string]int32)
	for _, t := range tags {
		tag, err := queries.CreateTag(ctx, t.name, sql.NullString{String: t.description, Valid: true})
		if err != nil {
			log.Fatal(err)
		}
		tagIDs[t.name] = tag.Id
		fmt.Printf("Created tag: %s\n", tag.Name)
	}

	// Create posts
	post1, err := queries.CreatePost(ctx, author1.Id, "Getting Started with SQLite in Go", "This is a comprehensive guide...", 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created post: %s\n", post1.Title)

	// Add tags to post
	if _, err := queries.AddTagToPost(ctx, post1.Id, tagIDs["go"]); err != nil {
		log.Fatal(err)
	}
	if _, err := queries.AddTagToPost(ctx, post1.Id, tagIDs["sqlite"]); err != nil {
		log.Fatal(err)
	}
	if _, err := queries.AddTagToPost(ctx, post1.Id, tagIDs["tutorial"]); err != nil {
		log.Fatal(err)
	}

	// Create more posts
	post2, err := queries.CreatePost(ctx, author2.Id, "Advanced SQLite Optimization", "Deep dive into query optimization...", 1)
	if err != nil {
		log.Fatal(err)
	}

	// Add tags to post2
	if _, err := queries.AddTagToPost(ctx, post2.Id, tagIDs["sqlite"]); err != nil {
		log.Fatal(err)
	}
	if _, err := queries.AddTagToPost(ctx, post2.Id, tagIDs["advanced"]); err != nil {
		log.Fatal(err)
	}

	// Create unpublished post
	_, err = queries.CreatePost(ctx, author1.Id, "Draft: Upcoming Features", "This is still being written...", 0)
	if err != nil {
		log.Fatal(err)
	}

	// Simulate views
	for range 100 {
		if _, err := queries.IncrementViewCount(ctx, post1.Id); err != nil {
			log.Fatal(err)
		}
	}
	for range 50 {
		if _, err := queries.IncrementViewCount(ctx, post2.Id); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n--- Published Posts ---")
	posts, err := queries.ListPosts(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range posts {
		fmt.Printf("  %s by author %d (views: %d)\n", p.Title, p.AuthorId, p.ViewCount)
	}

	fmt.Println("\n--- Post with Tags ---")
	postWithTags, err := queries.GetPost(ctx, post1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", postWithTags.Title)
	fmt.Printf("  Author: %d\n", postWithTags.AuthorId)

	postTags, err := queries.GetPostTags(ctx, post1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Tags: ")
	for i, t := range postTags {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(t.Name)
	}
	fmt.Println()

	fmt.Println("\n--- Popular Tags ---")
	limit := int32(5) //nolint:mnd // Example query limit
	popularTags, err := queries.GetPopularTags(ctx, &limit)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range popularTags {
		fmt.Printf("  %s: %d posts\n", t.Name, t.PostCount)
	}

	fmt.Println("\n--- Author Stats ---")
	stats, err := queries.GetAuthorStats(ctx, author1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  %s: %d total posts\n", stats.Name, stats.TotalPosts)

	fmt.Println("\n--- Unpublished Posts ---")
	unpublished, err := queries.ListUnpublishedPosts(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range unpublished {
		fmt.Printf("  %s by author %d\n", p.Title, p.AuthorId)
	}

	fmt.Println("\n--- Search Posts ---")
	searchResults, err := queries.SearchPosts(ctx, nil, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range searchResults {
		fmt.Printf("  %s by author %d\n", p.Title, p.AuthorId)
	}

	fmt.Println("\n--- Posts by Tag 'sqlite' ---")
	taggedPosts, err := queries.GetPostsByTag(ctx, "sqlite")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range taggedPosts {
		fmt.Printf("  %s by author %d\n", p.Title, p.AuthorId)
	}

	fmt.Println("\nSuccess! All complex queries executed.")
}

const schema = `
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    bio TEXT,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    published INTEGER NOT NULL DEFAULT 0,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE post_tags (
    post_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, tag_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_published ON posts(published);
CREATE INDEX idx_post_tags_post ON post_tags(post_id);
CREATE INDEX idx_post_tags_tag ON post_tags(tag_id);
`
