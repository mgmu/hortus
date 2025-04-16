package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

var (
	noUrl = "Database url is not set\n"
	connPoolErr = "Unable to create connection pool: %v\n"
	queryRowErr = "QueryRow failed: %v\n"
	noTables = "Tables don't exist\n"
)

func main() {
	// Connection to database
	dburl := os.Getenv("HORTUS_DB_URL")
	if dburl == "" {
		fmt.Fprintf(os.Stderr, noUrl)
		os.Exit(1)
	}

	dbpool, err := pgxpool.New(context.Background(), dburl)
	if err != nil {
		fmt.Fprintf(os.Stderr, connPoolErr, err)
		os.Exit(1)
	}
	defer dbpool.Close()

	// Check existence of tables
	query := `SELECT EXISTS (
SELECT FROM pg_tables
WHERE schemaname = 'hortus_schema'
AND (tablename = 'plant' OR tablename = 'plant_log')
);`

	var exist bool
	err = dbpool.QueryRow(context.Background(), query).Scan(&exist)
	if err != nil {
		fmt.Fprintf(os.Stderr, queryRowErr, err)
		os.Exit(1)
	}

	if !exist {
		fmt.Fprintf(os.Stderr, noTables)
		os.Exit(1)
	}
}
