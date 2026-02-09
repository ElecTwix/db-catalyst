package graphql

import (
	"context"
	"testing"
)

// FuzzParser tests GraphQL parser with random inputs.
func FuzzParser(f *testing.F) {
	// Valid GraphQL schemas
	f.Add("type User { id: ID name: String }")
	f.Add("type Query { user(id: ID): User }")
	f.Add(`type Post {
		id: ID
		title: String
		content: String
		author: User
	}`)
	f.Add(`interface Node {
		id: ID
	}
	type User implements Node {
		id: ID
		name: String
	}`)
	f.Add(`enum Status {
		ACTIVE
		INACTIVE
	}`)
	f.Add(`input CreateUserInput {
		name: String
		email: String
	}`)
	f.Add(`type Mutation {
		createUser(input: CreateUserInput): User
	}`)
	// Edge cases
	f.Add("")
	f.Add("# comment")
	f.Add("type")
	f.Add("type User")
	f.Add("type User {")
	f.Add("type User { }")
	f.Add("type User { id }")
	f.Add("type User { id: }")
	// Unicode
	f.Add("type ユーザー { id: ID }")
	f.Add("type User { 名前: String }")
	// Malformed
	f.Add("type User { id: ID name: String }") // Missing newline
	f.Add("type 123 { id: ID }")               // Invalid type name
	f.Add("type User { 123: String }")         // Invalid field name

	f.Fuzz(func(t *testing.T, input string) {
		p, err := NewParser()
		if err != nil {
			// Parser creation should never fail in production, but might with fuzzing
			return
		}
		// Parser should never panic
		_, _ = p.ParseSchema(context.Background(), input)
	})
}

// FuzzParserValidate tests the validation function.
func FuzzParserValidate(f *testing.F) {
	f.Add("type User { id: ID name: String }")
	f.Add("type Query { users: [User] }")
	f.Add("")
	f.Add("invalid")

	f.Fuzz(func(t *testing.T, input string) {
		p, err := NewParser()
		if err != nil {
			return
		}
		// Should never panic
		_, _ = p.Validate(input)
	})
}
