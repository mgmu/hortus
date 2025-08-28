package database

import (
	"github.com/mgmu/hortus/internal/plants"
)

// Database defines the API to store and retrieve plants and related data from a
// database.
type Database interface {
	Connect() error
	Close() error
	GetPlantsShortDescription() ([]plants.PlantShortDesc, error)
	AddNewPlant(comm, gen, spe string) (int, error)
	GetPlantNames(id int) (string, string, string, error)
	GetPlantLogs(id int) ([]plants.PlantLog, error)
	AddNewPlantLog(id int, desc string, event int) error
}
