package main

import (
	"context"
	"fmt"
	"github.com/mgmu/hortus/api/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
)

var (
	noUrl         = "Database url is not set\n"
	connPoolErr   = "Unable to create connection pool: %v\n"
	queryRowErr   = "QueryRow failed: %v\n"
	noTables      = "Tables don't exist\n"
	searchPathErr = "Could not change search path\n"
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

	// Change schema (it should exist at this point)
	query = `SET search_path TO hortus_schema;`
	_, err = dbpool.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Add API handlers
	http.HandleFunc("/plants/", handlers.PlantsListHandler(dbpool))
	http.HandleFunc("/plants/new/", handlers.NewPlantHandler(dbpool))
	http.HandleFunc("/plants/{id}/", handlers.PlantInfoHandler(dbpool))
	http.HandleFunc("/plants/log/{id}/", handlers.NewPlantLogHandler(dbpool))

	// Start server
	err = http.ListenAndServe(":8080", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}
