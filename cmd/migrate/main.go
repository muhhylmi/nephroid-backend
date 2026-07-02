package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"backend/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:password@localhost:5432/nephroid?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Read files from the embedded FS
	files, err := db.MigrationFiles.ReadDir(".")
	if err != nil {
		fmt.Printf("Failed to read migration files: %v\n", err)
		os.Exit(1)
	}

	var fileNames []string
	for _, f := range files {
		if !f.IsDir() {
			fileNames = append(fileNames, f.Name())
		}
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		content, err := db.MigrationFiles.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", fileName, err)
			os.Exit(1)
		}

		fmt.Printf("Running migration: %s\n", fileName)
		_, err = pool.Exec(context.Background(), string(content))
		if err != nil {
			fmt.Printf("Migration failed on %s: %v\n", fileName, err)
			os.Exit(1)
		}
	}

	fmt.Println("All migrations applied successfully!")
}
