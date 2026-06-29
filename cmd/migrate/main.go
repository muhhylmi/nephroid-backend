package main

import (
	"context"
	"fmt"
	"os"

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
		panic(err)
	}
	defer pool.Close()

	query := `
	ALTER TABLE users ADD COLUMN IF NOT EXISTS target_dry_weight NUMERIC DEFAULT 60.0;

	CREATE TABLE IF NOT EXISTS weight_records (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		date VARCHAR(50) NOT NULL,
		pre_weight NUMERIC NOT NULL,
		post_weight NUMERIC NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS lab_records (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		date VARCHAR(50) NOT NULL,
		kreatinin NUMERIC NOT NULL,
		ureum NUMERIC NOT NULL,
		kalium NUMERIC NOT NULL,
		hb NUMERIC NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = pool.Exec(context.Background(), query)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Success")
}
