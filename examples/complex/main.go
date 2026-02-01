package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/electwix/db-catalyst/examples/complex/db"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	// Open database
	sqlDB, err := sql.Open("modernc.org/sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema
	if _, err := sqlDB.Exec(schema); err != nil {
		log.Fatal(err)
	}

	queries := db.New(sqlDB)

	// Create authors
	author1, err := queries.CreateAuthor(ctx, db.CreateAuthorParams{
		Name:  "Alice Smith",
		Email: "alice@example.com",
		Bio:   sql.NullString{String: "Go enthusiast", Valid: true},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created author: %s\n", author1.Name)

	author2, err := queries.CreateAuthor(ctx, db.CreateAuthorParams{
		Name:  "Bob Jones",
		Email: "bob@example.com",
		Bio:   sql.NullString{String: "SQLite expert", Valid: true},
	})
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

	tagIDs := make(map[string]int64)
	for _, t := range tags {
		tag, err := queries.CreateTag(ctx, db.CreateTagParams{
			Name:        t.name,
			Description: sql.NullString{String: t.description, Valid: true},
		})
		if err != nil {
			log.Fatal(err)
		}
		tagIDs[t.name] = tag.ID
		fmt.Printf("Created tag: %s\n", tag.Name)
	}

	// Create posts
	post1, err := queries.CreatePostWithTags(ctx, db.CreatePostWithTagsParams{
		AuthorID:  author1.ID,
		Title:     "Getting Started with SQLite in Go",
		Content:   "This is a comprehensive guide...",
		Published: 1,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created post: %s\n", post1.Title)

	// Add tags to post
	if err := queries.AddTagToPost(ctx, db.AddTagToPostParams{
		PostID: post1.ID,
		TagID:  tagIDs["go"],
	}); err != nil {
		log.Fatal(err)
	}
	if err := queries.AddTagToPost(ctx, db.AddTagToPostParams{
		PostID: post1.ID,
		TagID:  tagIDs["sqlite"],
	}); err != nil {
		log.Fatal(err)
	}
	if err := queries.AddTagToPost(ctx, db.AddTagToPostParams{
		PostID: post1.ID,
		TagID:  tagIDs["tutorial"],
	}); err != nil {
		log.Fatal(err)
	}

	// Create more posts
	post2, err := queries.CreatePostWithTags(ctx, db.CreatePostWithTagsParams{
		AuthorID:  author2.ID,
		Title:     "Advanced SQLite Optimization",
		Content:   "Deep dive into query optimization...",
		Published: 1,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Add tags to post2
	if err := queries.AddTagToPost(ctx, db.AddTagToPostParams{
		PostID: post2.ID,
		TagID:  tagIDs["sqlite"],
	}); err != nil {
		log.Fatal(err)
	}
	if err := queries.AddTagToPost(ctx, db.AddTagToPostParams{
		PostID: post2.ID,
		TagID:  tagIDs["advanced"],
	}); err != nil {
		log.Fatal(err)
	}

	// Create unpublished post
	post3, err := queries.CreatePostWithTags(ctx, db.CreatePostWithTagsParams{
		AuthorID:  author1.ID,
		Title:     "Draft: Upcoming Features",
		Content:   "This is still being written...",
		Published: 0,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Simulate views
	for i := 0; i < 100; i++ {
		if err := queries.IncrementViewCount(ctx, post1.ID); err != nil {
			log.Fatal(err)
		}
	}
	for i := 0; i < 50; i++ {
		if err := queries.IncrementViewCount(ctx, post2.ID); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n--- Published Posts ---")
	posts, err := queries.ListPostsWithAuthor(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range posts {
		fmt.Printf("  %s by %s (views: %d)\n", p.Title, p.AuthorName, p.ViewCount)
	}

	fmt.Println("\n--- Post with Tags ---")
	postWithTags, err := queries.GetPostWithTags(ctx, post1.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", postWithTags.Title)
	fmt.Printf("  Author: %s\n", postWithTags.AuthorName)
	fmt.Printf("  Tags: %s\n", postWithTags.Tags)

	fmt.Println("\n--- Popular Tags ---")
	popularTags, err := queries.GetPopularTags(ctx, 5)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range popularTags {
		fmt.Printf("  %s: %d posts\n", t.Name, t.PostCount)
	}

	fmt.Println("\n--- Author Stats ---")
	stats, err := queries.GetAuthorStats(ctx, author1.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  %s: %d total, %d published, %d views\n",
		stats.Name, stats.TotalPosts, stats.PublishedCount, stats.TotalViews)

	fmt.Println("\n--- Unpublished Posts ---")
	unpublished, err := queries.ListUnpublishedPosts(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range unpublished {
		fmt.Printf("  %s by %s\n", p.Title, p.AuthorName)
	}

	fmt.Println("\n--- Search Posts ---")
	searchResults, err := queries.SearchPosts(ctx, db.SearchPostsParams{
		Column1: "SQLite",
		Column2: "SQLite",
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range searchResults {
		fmt.Printf("  %s by %s\n", p.Title, p.AuthorName)
	}

	fmt.Println("\n--- Posts by Tag 'sqlite' ---")
	taggedPosts, err := queries.GetPostsByTag(ctx, "sqlite")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range taggedPosts {
		fmt.Printf("  %s by %s\n", p.Title, p.AuthorName)
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
