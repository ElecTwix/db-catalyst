package mysql

import (
	"context"
	"testing"
)

// FuzzParser tests MySQL parser with random inputs.
func FuzzParser(f *testing.F) {
	// Seed corpus with valid MySQL DDL
	f.Add("CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY);")
	f.Add("CREATE TABLE users (id BIGINT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL);")
	f.Add("CREATE TABLE products (id INT AUTO_INCREMENT PRIMARY KEY, status ENUM('active', 'inactive') DEFAULT 'active');")
	f.Add("CREATE TABLE tags (id INT AUTO_INCREMENT PRIMARY KEY, flags SET('a', 'b', 'c') DEFAULT 'a');")
	f.Add("CREATE TABLE events (id BIGINT AUTO_INCREMENT PRIMARY KEY, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP);")
	f.Add("CREATE TABLE logs (id INT AUTO_INCREMENT PRIMARY KEY, message TEXT, FULLTEXT INDEX ft_idx (message));")
	f.Add("CREATE TABLE spatial (id INT PRIMARY KEY, location POINT NOT NULL, SPATIAL INDEX sp_idx (location));")
	f.Add("CREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, email VARCHAR(255) UNIQUE KEY uk_email (email));")
	f.Add("CREATE TABLE orders (id INT AUTO_INCREMENT PRIMARY KEY, user_id INT, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE);")
	// Edge cases
	f.Add("")
	f.Add("-- MySQL comment")
	f.Add("# Hash comment")
	f.Add("CREATE TABLE")
	f.Add("CREATE TABLE t") // Missing columns
	f.Add("CREATE TABLE t ()")
	// Unicode
	f.Add("CREATE TABLE 用户 (id INT PRIMARY KEY);")
	f.Add("CREATE TABLE users (名称 VARCHAR(100));")
	// Malformed
	f.Add("CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY;")
	f.Add("CREATE TABLE t (status ENUM('a',))")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		// Parser should never panic
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}

// FuzzParserWithStorageEngines tests different storage engine syntax.
func FuzzParserWithStorageEngines(f *testing.F) {
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=MyISAM;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=MEMORY;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB COLLATE=utf8mb4_unicode_ci;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB AUTO_INCREMENT=100;")
	f.Add("CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB ROW_FORMAT=DYNAMIC;")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}

// FuzzParserWithColumnAttributes tests various MySQL column attributes.
func FuzzParserWithColumnAttributes(f *testing.F) {
	f.Add("CREATE TABLE t (id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY);")
	f.Add("CREATE TABLE t (id INT ZEROFILL);")
	f.Add("CREATE TABLE t (id INT UNSIGNED ZEROFILL);")
	f.Add("CREATE TABLE t (name VARCHAR(100) CHARACTER SET utf8mb4);")
	f.Add("CREATE TABLE t (name VARCHAR(100) COLLATE utf8mb4_bin);")
	f.Add("CREATE TABLE t (name VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci);")
	f.Add("CREATE TABLE t (comment TEXT COMMENT 'User comment');")
	f.Add("CREATE TABLE t (id INT INVISIBLE);")
	f.Add("CREATE TABLE t (id INT STORED GENERATED ALWAYS AS (id * 2));")
	f.Add("CREATE TABLE t (id INT VIRTUAL GENERATED ALWAYS AS (id * 2));")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}
