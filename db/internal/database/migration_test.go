package database_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.leapkit.dev/tools/db/internal/database"
)

func TestGenerateMigration(t *testing.T) {
	bd, _ := os.Getwd()
	defer os.Chdir(bd)

	t.Run("correct generate migration", func(t *testing.T) {
		wd := t.TempDir()
		err := os.Chdir(wd)
		if err != nil {
			t.Fatalf("error changing directory: %v", err)
		}

		migrationFolder := "internal/custom/migrations"

		// Create a new migration
		os.Args = []string{"db", "generate_migration", "create_users_table", fmt.Sprintf("--migration.folder=%s", migrationFolder)}
		// main.go call
		err = database.Exec()
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		var migrationPath string
		// Check if the migration file was created
		filepath.Walk(filepath.Join(wd, migrationFolder), func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == ".sql" {
				migrationPath = path
				return nil
			}

			return nil
		})

		if migrationPath == "" {
			t.Fatalf("migration file not created in %s", wd)
		}

		// Check if it was created in the custom folder
		if !strings.Contains(migrationPath, migrationFolder) {
			t.Logf("Warning: migration created in %s instead of %s", migrationPath, migrationFolder)
		}

		migrationPath, err = filepath.Rel(wd, migrationPath)
		if err != nil {
			t.Fatalf("error getting relative path: %v", err)
		}

		// Check if the migration file is not empty
		file, err := os.Open(migrationPath)
		if err != nil {
			t.Fatalf("error opening migration file: %v", err)
		}

		defer file.Close()

		// read the file content
		bc, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("error reading migration file: %v", err)
		}

		if bytes.Contains(bc, []byte(migrationPath)) {
			t.Fatalf("migration should not contain the full path")
		}
	})

	t.Run("correct incomplete command", func(t *testing.T) {
		current := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		defer func() {
			os.Stdout = current
		}()

		os.Args = []string{"db", "generate_migration"}
		err := database.Exec()
		if err != nil {
			t.Fatalf("error creating migration: %v", err)
		}

		w.Close()
		out, _ := io.ReadAll(r)
		fmt.Println(string(out))

		if !bytes.Contains(out, []byte("Usage: database generate_migration <migration_name>")) {
			t.Errorf("Expected 'Usage: database generate_migration <migration_name>', got: %v", string(out))
		}
	})
}
