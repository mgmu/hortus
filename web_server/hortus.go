package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var (
	protocol      = "http"
	webPort       = "8081"
	apiPort       = "8080"
	plantsListUrl = "/plants/"
	newPlantUrl   = plantsListUrl + "new/"
	notAllowed    = "Method not allowed"
)

// Encapsulate the common name of a plant and a link to the web page displaying
// more detailed information.
type plantLink struct {
	Link       string
	CommonName string
}

// Encapsulates environment data for URL handlers
type env struct {
	t         *template.Template
	hortusWeb string // URL for hortus web site
	hortusApi string // URL for hortus api
}

// jsonPlant describes a plant as a json object.
// This is used by plantInfoHandler to send the plant information as json
// encoded data.
type jsonPlant struct {
	Id           int            `json:"id"`
	CommonName   string         `json:"common_name"`
	GenericName  string         `json:"generic_name"`
	SpecificName string         `json:"specific_name"`
	Logs         []jsonPlantLog `json:"logs"`
}

// jsonPlantLog describes a plant log as a json object.
type jsonPlantLog struct {
	Id        int    `json:"id"`
	PlantId   int    `json:"plant_id"`
	Desc      string `json:"desc"`
	EventType int    `json:"event_type"`
}

// jsonPlantShortDesc type encapsulates the short description of a plant: its
// identifier and common name.
type jsonPlantShortDesc struct {
	Id         int    `json:"id"`
	CommonName string `json:"common_name"`
}

func main() {
	hortusWebIp := os.Getenv("HORTUS_WEB_IP")
	if hortusWebIp == "" {
		fmt.Fprintf(os.Stderr, "No IP provided for Hortus web\n")
		os.Exit(1)
	}

	hortusApiIp := os.Getenv("HORTUS_API_IP")
	if hortusApiIp == "" {
		fmt.Fprintf(os.Stderr, "No IP provided for Hortus API\n")
		os.Exit(1)
	}

	t, err := template.ParseFiles(
		"templates/meta-tags.gohtml",
		"templates/nav-bar.gohtml",
		"templates/index.gohtml",
		"templates/newPlant.gohtml",
		"templates/plantInfo.gohtml",
		"templates/newPlantLog.gohtml",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	e := env{
		t,
		protocol + "://" + hortusWebIp + ":" + webPort,
		protocol + "://" + hortusApiIp + ":" + apiPort,
	}
	http.HandleFunc("/", e.indexHandler())
	http.HandleFunc("/plants/new/", e.newPlantHandler())
	http.HandleFunc("/plants/{id}/", e.plantInfoHandler())
	http.HandleFunc("/plants/log/{id}/", e.newPlantLogHandler())

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
		resp, err := http.Get(e.hortusApi + plantsListUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		var plants []jsonPlantShortDesc
		err = dec.Decode(&plants)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		plantLinks := plantsShortDescToPlantLinks(plants, e.hortusWeb)

		// Send HTML document
		err = e.t.ExecuteTemplate(w, "index.gohtml", plantLinks)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

/* Returns a handler for the "/plants/new" URL.
 * The request method should be either GET or POST. If it is GET, returns an
 * html page with a form to add a new plant. The submit button sends a POST
 * request to the same URL. If the request method is POST, parses the form and
 * sends a POST request to the API to add the new plant, redirecting to the
 * plant's information page.
 */
func (e *env) newPlantHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		if r.Method == http.MethodGet {
			err := e.t.ExecuteTemplate(w, "newPlant.gohtml", nil)
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
			url := e.hortusApi + newPlantUrl
			resp, err := http.PostForm(url, data)
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

			url = e.hortusWeb + plantsListUrl + id
			http.Redirect(
				w,
				r,
				url,
				http.StatusSeeOther,
			)
		}
	}
}

/* Returns a handler for the "/plants/{id}" URL.
 * The request method should be GET. On success, returns an html document with
 * relevant information on the plant requested. If the requested plant
 * identifier is not a number, an error response is sent. This handler sends
 * a GET request to the API to fetch the relevant plant information, if an error
 * occurs, the handler sends back to the client an error response.
 */
func (e *env) plantInfoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		url := e.hortusApi + plantsListUrl + r.PathValue("id")
		resp, err := http.Get(url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		var plantInfo jsonPlant
		err = dec.Decode(&plantInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = e.t.ExecuteTemplate(w, "plantInfo.gohtml", plantInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

/* Returns a handler for the URL "/plants/log/{id}/".
 */
func (e *env) newPlantLogHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			err = e.t.ExecuteTemplate(w, "newPlantLog.gohtml", id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodPost:
			err = r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data := url.Values{}
			data.Set("new-entry", r.PostForm.Get("new-entry"))
			url := e.hortusApi + "/plants/log/" + r.PathValue("id") + "/"
			resp, err := http.PostForm(url, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			url = e.hortusWeb + plantsListUrl + r.PathValue("id") + "/"
			http.Redirect(
				w,
				r,
				url,
				http.StatusSeeOther,
			)
		default:
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}
	}
}

// converts a slice of plant short descriptions to a slice of plant links
func plantsShortDescToPlantLinks(
	psd []jsonPlantShortDesc,
	link string,
) []plantLink {
	plantLinks := make([]plantLink, len(psd))
	for i, plant := range psd {
		plantLinks[i].Link =
			link + plantsListUrl + strconv.Itoa(plant.Id)
		plantLinks[i].CommonName = plant.CommonName
	}
	return plantLinks
}
