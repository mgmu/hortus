package plants

// PlantShortDesc type encapsulates the short description of a plant: its
// identifier and common name.
type PlantShortDesc struct {
	Id         int    `json:"id"`
	CommonName string `json:"common_name"`
}

// Represents a plant by its name, its scientific name and its log entries.
type Plant struct {
	Id           int        `json:"id"`
	CommonName   string     `json:"common_name"`
	GenericName  string     `json:"generic_name"`
	SpecificName string     `json:"specific_name"`
	Logs         []PlantLog `json:"logs"`
}

// Represents a plant log by the plant to wich it belongs, its identifier, its
// description and its type
type PlantLog struct {
	Id        int    `json:"id"`
	PlantId   int    `json:"plant_id"`
	Desc      string `json:"desc"`
	EventType int    `json:"event_type"`
}
