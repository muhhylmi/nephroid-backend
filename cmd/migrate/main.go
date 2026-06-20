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

	content, err := os.ReadFile("./db/init.sql")
	if err != nil {
		panic(err)
	}

	_, err = pool.Exec(context.Background(), string(content))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Success")
}
