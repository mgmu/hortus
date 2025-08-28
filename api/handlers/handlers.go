package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mgmu/hortus/api/database"
	"github.com/mgmu/hortus/internal/plants"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	notAllowed = "Method not allowed"
	nameMaxLen = 255
)

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
func PlantsListHandler(db database.Database) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		if method != http.MethodHead && method != http.MethodGet {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		plants, err := db.GetPlantsShortDescription()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Method == http.MethodGet {
			enc := json.NewEncoder(w)
			err = enc.Encode(plants)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

/* Returns a handler for the "/plants/new" URL.
 * The request method should be POST. If it is not, sets the status code to
 * http.StatusMethodNotAllowed and sends an error response. If an error is
 * encountered when calling ParseForm or inserting the new plant , sends a
 * "Bad Request" error back. Otherwise, the new plant is inserted and its
 * identifier is sent back in the body in its textual form.
 */
func NewPlantHandler(db database.Database) func(http.ResponseWriter, *http.Request) {
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

		id, err := db.AddNewPlant(comm, gen, spe)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(strconv.Itoa(id))))
		fmt.Fprintf(w, strconv.Itoa(id))
	}
}

/* Returns a handler for the "/plants/{id}" URL.
 * The request method should be GET or HEAD. If it is not, sets the status code
 * to http.StatusMethodNotAllowed and sends an error response. Queries the
 * database for plant information and sends it back as json encoded data.
 */
func PlantInfoHandler(db database.Database) func(http.ResponseWriter, *http.Request) {
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

		comm, gen, spe, err := db.GetPlantNames(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		plantLogs, err := db.GetPlantLogs(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Encode the plant as a json object
		plant := plants.Plant{id, comm, gen, spe, plantLogs}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Method == http.MethodGet {
			err = json.NewEncoder(w).Encode(plant)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func NewPlantLogHandler(db database.Database) func(http.ResponseWriter, *http.Request) {
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

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = db.AddNewPlantLog(id, r.PostForm.Get("new-entry"), 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		return
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
