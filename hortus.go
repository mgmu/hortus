package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	hortusApi     = "http://localhost:8080"
	hortusWeb     = "http://localhost:8081"
	plantsListUrl = "/plants/"
	newPlantUrl   = plantsListUrl + "new/"
)

func main() {
	http.HandleFunc("/", indexHandler)

	err := http.ListenAndServe(":8081", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}

type plantLink struct {
	Link       string
	CommonName string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Load html template
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
	err = t.Execute(w, plants)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

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
