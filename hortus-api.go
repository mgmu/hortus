package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	noUrl         = "Database url is not set\n"
	connPoolErr   = "Unable to create connection pool: %v\n"
	queryRowErr   = "QueryRow failed: %v\n"
	noTables      = "Tables don't exist\n"
	searchPathErr = "Could not change search path\n"
	notAllowed    = "Method not allowed"
	emptyName     = "Name is empty"
	longName      = "Name is too long"
	illegalName   = "Name contains an illegal character"
	nameMaxLen    = 255
)

// env type encapsulates the database connection pool, needed by the URL
// handlers of the API.
type env struct {
	conn *pgxpool.Pool
}

// plantShortDesc type encapsulates the short description of a plant: its
// identifier and common name.
type plantShortDesc struct {
	Id         int
	CommonName string
}

// jsonPlant describes a plant as a json object.
// This is used by plantInfoHandler to send the plant information as json
// encoded data.
type jsonPlant struct {
	Id           int    `json:"id"`
	CommonName   string `json:"commonName"`
	GenericName  string `json:"genericName"`
	SpecificName string `json:"specificName"`
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
	http.HandleFunc("/plants/new/", e.newPlantHandler())
	http.HandleFunc("/plants/{id}/", e.plantInfoHandler())

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
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		// Query the db for identifiers and plant names
		query := "SELECT id, common_name FROM plant;"
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

/*
 * Returns a handler for the "/plants/new" URL.
 * The request method should be POST. If it is not, sets the status code to
 * http.StatusMethodNotAllowed and sends an error response. If an error is
 * encountered when calling ParseForm or inserting the new plant , sends a
 * "Bad Request" error back. Otherwise, the new plant is inserted and its
 * identifier is sent back in the body in its textual form.
 */
func (e *env) newPlantHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate input
		comm, err := sanitizeCommonName(r.PostForm.Get("common-name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		gen, err := sanitizeScientificName(r.PostForm.Get("generic-name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		spe, err := sanitizeScientificName(r.PostForm.Get("specific-name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Insert new plant
		row := e.conn.QueryRow(
			context.Background(),
			`
INSERT INTO plant (common_name, generic_name, specific_name)
VALUES ($1, $2, $3)
RETURNING id;`,
			comm,
			gen,
			spe,
		)
		var id int
		err = row.Scan(&id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(strconv.Itoa(id))))

		fmt.Fprintf(w, strconv.Itoa(id))
	}
}

func (e *env) plantInfoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		// Get the id of the plant to fetch from the url
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the db for the plant
		row := e.conn.QueryRow(
			context.Background(),
			`SELECT * FROM plant WHERE id=$1;`,
			id,
		)
		var comm, gen, spe string
		err = row.Scan(&id, &comm, &gen, &spe)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Encode the plant as a json object
		plant := jsonPlant{id, comm, gen, spe}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Method == http.MethodGet {
			enc := json.NewEncoder(w)
			err = enc.Encode(plant)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

// Checks that name is not empty after trim, not longer than 255
// characters and valid utf8. The string returned is the trimmed version of
// common name.
func sanitizeCommonName(name string) (string, error) {
	s := strings.TrimSpace(name)
	if len(s) == 0 {
		return "", errors.New("Common name is empty")
	}
	if len(s) > nameMaxLen {
		return "", errors.New("Common name length is greater than 255")
	}
	if !utf8.ValidString(s) {
		return "", errors.New("Common name is not UTF-8")
	}
	return s, nil
}

// Checks that name is not longer than 255 characters after trim and is ascii.
// The string returned is the trimmed version of name.
func sanitizeScientificName(name string) (string, error) {
	s := strings.TrimSpace(name)
	if len(s) > nameMaxLen {
		return "", errors.New("Scientific name length is greater than 255")
	}
	if !isAscii(s) {
		return "", errors.New("Specific name is not ASCII")
	}
	return s, nil
}

// Returns true if s contains only ASCII characters.
func isAscii(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}
