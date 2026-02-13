package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	simpledb "github.com/electwix/db-catalyst/examples/simple/db"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	// Open SQLite database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema
	if _, err := sqlDB.Exec(schema); err != nil {
		log.Fatal(err)
	}

	// Create querier
	queries := simpledb.New(sqlDB)

	// Create a user
	user, err := queries.CreateUser(ctx, "John Doe", sql.NullString{String: "john@example.com", Valid: true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created user: %+v\n", user)

	// Get the user
	fetched, err := queries.GetUser(ctx, user.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Fetched user: %+v\n", fetched)

	// List all users
	users, err := queries.ListUsers(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total users: %d\n", len(users))

	// Update user
	updated, err := queries.UpdateUser(ctx, "Jane Doe", sql.NullString{String: "jane@example.com", Valid: true}, user.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated user: %+v\n", updated)

	// Delete user
	_, err = queries.DeleteUser(ctx, user.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User deleted")

	// Verify deletion
	_, err = queries.GetUser(ctx, user.Id)
	if err == sql.ErrNoRows {
		fmt.Println("User not found (expected)")
	}
}

const schema = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
`
