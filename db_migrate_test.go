package goo

import (
	"log/slog"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
	"github.com/stretchr/testify/assert"
)

func TestMigrator(t *testing.T) {
	assert := assert.New(t)

	// Create an in-memory SQLite database for testing
	db, err := sqlx.Open("sqlite3", ":memory:")
	assert.NoError(err)
	defer db.Close()

	// Create a test logger that writes to nowhere
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create a new migrator
	migrator := ProvideDBMigrator(db, logger)
	assert.NotNil(migrator)

	// Define test migrations
	migrations := []Migration{
		{
			Name: "create_users_table",
			Up: `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					email TEXT NOT NULL UNIQUE
				);
			`,
		},
		{
			Name: "create_posts_table",
			Up: `
				CREATE TABLE posts (
					id INTEGER PRIMARY KEY,
					user_id INTEGER NOT NULL,
					title TEXT NOT NULL,
					content TEXT NOT NULL,
					FOREIGN KEY (user_id) REFERENCES users(id)
				);
			`,
		},
	}

	// Run migrations
	err = migrator.Up(migrations)
	assert.NoError(err)

	// Verify that migrations table exists and contains our migrations
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM migrations")
	assert.NoError(err)
	assert.Equal(2, count)

	// Verify that users table was created
	err = db.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'")
	assert.NoError(err)
	assert.Equal(1, count)

	// Verify that posts table was created
	err = db.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='posts'")
	assert.NoError(err)
	assert.Equal(1, count)

	// Run migrations again - should be idempotent
	err = migrator.Up(migrations)
	assert.NoError(err)

	// Should still have only 2 migrations in the table
	err = db.Get(&count, "SELECT COUNT(*) FROM migrations")
	assert.NoError(err)
	assert.Equal(2, count)

	// Add a new migration and run again
	newMigrations := append(migrations, Migration{
		Name: "create_comments_table",
		Up: `
			CREATE TABLE comments (
				id INTEGER PRIMARY KEY,
				post_id INTEGER NOT NULL,
				content TEXT NOT NULL,
				FOREIGN KEY (post_id) REFERENCES posts(id)
			);
		`,
	})

	err = migrator.Up(newMigrations)
	assert.NoError(err)

	// Should now have 3 migrations in the table
	err = db.Get(&count, "SELECT COUNT(*) FROM migrations")
	assert.NoError(err)
	assert.Equal(3, count)

	// Verify that comments table was created
	err = db.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='comments'")
	assert.NoError(err)
	assert.Equal(1, count)
}
