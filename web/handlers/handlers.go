package handlers

import (
	"encoding/json"
	"github.com/mgmu/hortus/internal/plants"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

var (
	IndexRoute       = "/"
	NewPlantRoute    = "/plants/new/"
	PlantInfoRoute   = "/plants/{id}/"
	NewPlantLogRoute = "/plants/log/{id}/"
	plantsListUrl    = "/plants/"
	notAllowed       = "Method not allowed"
)

// Encapsulates environment data for URL handlers
type HandlerEnv struct {
	templates *template.Template
	webUrl    string
	apiUrl    string
	navBar    navBarLinks
}

func New(webUrl, apiUrl string) (HandlerEnv, error) {
	t, err := template.ParseFiles(
		"templates/meta-tags.gohtml",
		"templates/nav-bar.gohtml",
		"templates/index.gohtml",
		"templates/newPlant.gohtml",
		"templates/plantInfo.gohtml",
		"templates/newPlantLog.gohtml",
	)
	if err != nil {
		return HandlerEnv{}, err
	}
	navBar := navBarLinks{webUrl + "/", webUrl + "/plants/new/"}
	return HandlerEnv{t, webUrl, apiUrl, navBar}, nil
}

// Encapsulates the nav bar links
type navBarLinks struct {
	Home     string
	AddPlant string
}

// Encapsulates the common name of a plant and a link to the web page displaying
// more detailed information.
type plantLink struct {
	Link       string
	CommonName string
}

type plantLinksWithNavBar struct {
	PlantLinks []plantLink
	NavBar     navBarLinks
}

type plantInfoWithNavBar struct {
	Plant  plants.Plant
	NavBar navBarLinks
}

type plantIdWithNavBar struct {
	Id     int
	NavBar navBarLinks
}

// Returns a handler for the "/" or "/index.html" URL.
// The request method should be GET. The handler sends a GET request to the API
// that fetches the plants list and sends back to the client a HTML document
// with the plants list.
func (e *HandlerEnv) IndexHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Fetch plants
		resp, err := http.Get(e.apiUrl + plantsListUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		var plants []plants.PlantShortDesc
		err = dec.Decode(&plants)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		links := plantsShortDescToPlantLinks(plants, e.webUrl)
		linksWithNav := plantLinksWithNavBar{links, e.navBar}

		// Send HTML document
		err = e.templates.ExecuteTemplate(w, "index.gohtml", linksWithNav)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Returns a handler for the "/plants/new" URL.
// The request method should be either GET or POST. If it is GET, returns an
// html page with a form to add a new plant. The submit button sends a POST
// request to the same URL. If the request method is POST, parses the form and
// sends a POST request to the API to add the new plant, redirecting to the
// plant's information page.
func (e *HandlerEnv) NewPlantHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		if r.Method == http.MethodGet {
			err := e.templates.ExecuteTemplate(w, "newPlant.gohtml", e.navBar)
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
			url := e.apiUrl + plantsListUrl + "new/"
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

			url = e.webUrl + plantsListUrl + id
			http.Redirect(
				w,
				r,
				url,
				http.StatusSeeOther,
			)
		}
	}
}

// Returns a handler for the "/plants/{id}" URL.
// The request method should be GET. On success, returns an html document with
// relevant information on the plant requested. If the requested plant
// identifier is not a number, an error response is sent. This handler sends
// a GET request to the API to fetch the relevant plant information, if an error
// occurs, the handler sends back to the client an error response.
func (e *HandlerEnv) PlantInfoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, notAllowed, http.StatusMethodNotAllowed)
			return
		}

		url := e.apiUrl + plantsListUrl + r.PathValue("id")
		resp, err := http.Get(url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		var plantInfo plants.Plant
		err = dec.Decode(&plantInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = e.templates.ExecuteTemplate(w, "plantInfo.gohtml", plantInfoWithNavBar{plantInfo, e.navBar})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Returns a handler for the URL "/plants/log/{id}/".
func (e *HandlerEnv) NewPlantLogHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			err = e.templates.ExecuteTemplate(w, "newPlantLog.gohtml", plantIdWithNavBar{id, e.navBar})
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
			url := e.apiUrl + "/plants/log/" + r.PathValue("id") + "/"
			resp, err := http.PostForm(url, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			url = e.webUrl + plantsListUrl + r.PathValue("id") + "/"
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
	psd []plants.PlantShortDesc,
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
