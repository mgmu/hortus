package main

import (
	"fmt"
	"github.com/mgmu/hortus/web/handlers"
	"log"
	"net/http"
	"os"
)

var (
	protocol = "http"
	webPort  = "8081"
	apiPort  = "8080"
)

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

	hortusWeb := protocol + "://" + hortusWebIp + ":" + webPort
	hortusApi := protocol + "://" + hortusApiIp + ":" + apiPort
	env, err := handlers.New(hortusWeb, hortusApi)
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc(handlers.IndexRoute, env.IndexHandler())
	http.HandleFunc(handlers.NewPlantRoute, env.NewPlantHandler())
	http.HandleFunc(handlers.PlantInfoRoute, env.PlantInfoHandler())
	http.HandleFunc(handlers.NewPlantLogRoute, env.NewPlantLogHandler())

	err = http.ListenAndServe(":8081", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}
