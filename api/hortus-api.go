package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	"strconv"
)

var (
	noUrl         = "Database url is not set\n"
	connPoolErr   = "Unable to create connection pool: %v\n"
	queryRowErr   = "QueryRow failed: %v\n"
	noTables      = "Tables don't exist\n"
	searchPathErr = "Could not change search path\n"
)

// env type encapsulates the database connection pool, needed by the URL
// handlers of the API.
type env struct {
	conn *pgxpool.Pool
}

// plantShortDesc type encapsulates the short description of plant: its
// identifier and its common name.
type plantShortDesc struct {
	Id         int
	CommonName string
}

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
	e := env{dbpool}
	http.HandleFunc("/plants/", e.plantsListHandler())

	// Start server
	err = http.ListenAndServe(":8080", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}

/* Returns a handler for the "/plants/" URL.
 * The request method should be either HEAD or GET. If the request method is
 * HEAD, sets the content length and returns 200 status code. If the request
 * method is GET, sets the content length, returns 200 status code and the body
 * of the response contains a list of plant name/identifier couples, separated
 * by a comma, one couple per line and each line ends with a new line character.
 * If the request method is not either HEAD or GET, sends a "Method not allowed"
 * error with the appropriate status code. If an error occurs while
 * communicating with the database, sends an "Internal Server Error" with the
 * appropriate status code and error message.
 */
func (e *env) plantsListHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		if method != http.MethodHead && method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Query the db for identifiers and plant names
		query := `SELECT id, common_name FROM plant;`
		rows, _ := e.conn.Query(context.Background(), query)
		plants, err := pgx.CollectRows(
			rows,
			pgx.RowToStructByPos[plantShortDesc],
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write response header and body
		var body string
		for i := 0; i < len(plants); i++ {
			body += strconv.Itoa(plants[i].Id)
			body += ","
			body += plants[i].CommonName
			body += "\n"
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))

		if method == http.MethodGet {
			fmt.Fprintf(w, body)
		}
	}
}
