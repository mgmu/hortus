package main

import (
	"fmt"
	"github.com/mgmu/hortus/api/database"
	"github.com/mgmu/hortus/api/handlers"
	"log"
	"net/http"
	"os"
)

func main() {
	// Connection to database
	db := database.PostgresDatabase{}
	err := db.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	// Add API handlers
	http.HandleFunc("/plants/", handlers.PlantsListHandler(&db))
	http.HandleFunc("/plants/new/", handlers.NewPlantHandler(&db))
	http.HandleFunc("/plants/{id}/", handlers.PlantInfoHandler(&db))
	http.HandleFunc("/plants/log/{id}/", handlers.NewPlantLogHandler(&db))

	// Start server
	err = http.ListenAndServe(":8080", nil)
	fmt.Fprintf(os.Stderr, "ListenAndServe: %v\n", err)
	os.Exit(1)
}
