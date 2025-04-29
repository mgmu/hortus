package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	hortusApi     = "http://localhost:8080"
	hortusWeb     = "http://localhost:8081"
	plantsListUrl = "/plants/"
	newPlantUrl   = plantsListUrl + "new/"
)

// Encapsulate the common name of a plant and a link to the web page displaying
// more detailed information.
type plantLink struct {
	Link       string
	CommonName string
}

// Encapsulates the template object for URL handlers
type env struct {
	t *template.Template
}

func main() {
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/newPlant.html",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	e := env{t}
	http.HandleFunc("/", e.indexHandler())
	http.HandleFunc("/plants/new/", e.newPlantHandler())

	err = http.ListenAndServe(":8081", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}

/* Returns a handler for the "/" or "/index.html" URL.
 * The request method should be GET. The handler sends a GET request to the API
 * that fetches the plants list and sends back to the client a HTML document
 * with the plants list.
 */
func (e *env) indexHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Fetch plants
		resp, err := http.Get(hortusApi + plantsListUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		plants, err := bytesToPlantLinks(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send HTML document
		err = e.t.ExecuteTemplate(w, "index.html", plants)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

/* Returns a handler for the "/plants/new" URL.
 * The request method should be either GET or POST. If it is GET, returns the
 * html page with a form to add a new plant. The submit button sends a POST
 * request to the same URL. If the request method is POST, parses the form and
 * sends a POST request to the API to add the new plant, redirecting to the
 * plant's information page.
 */
func (e *env) newPlantHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Method == http.MethodGet {
			err := e.t.ExecuteTemplate(w, "newPlant.html", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else { // POST
			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data := url.Values{}
			data.Set("common-name", r.PostForm.Get("common-name"))
			data.Set("generic-name", r.PostForm.Get("generic-name"))
			data.Set("specific-name", r.PostForm.Get("specific-name"))
			resp, err := http.PostForm(hortusApi + newPlantUrl, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			id := string(body)

			http.Redirect(
				w,
				r,
				hortusWeb + plantsListUrl + id,
				http.StatusSeeOther,
			)
		}
	}
}

// converts a slice of bytes into a slice of plantLink structures
func bytesToPlantLinks(b []byte) ([]plantLink, error) {
	var bd strings.Builder
	bd.Write(b) // Write always returns nil error
	text := bd.String()
	var plantLinks []plantLink
	if text == "" {
		return nil, nil
	}
	for line := range strings.Lines(text) {
		id, name, found := strings.Cut(line, ",")
		if !found {
			return nil, errors.New("Invalid list formatting")
		}
		link := hortusWeb + plantsListUrl + id
		plantLinks = append(plantLinks, plantLink{link, name})
	}
	return plantLinks, nil
}
