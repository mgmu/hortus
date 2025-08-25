package messages

// jsonPlantShortDesc type encapsulates the short description of a plant: its
// identifier and common name.
type JsonPlantShortDesc struct {
	Id         int    `json:"id"`
	CommonName string `json:"common_name"`
}

// jsonPlant describes a plant as a json object.
// This is used by plantInfoHandler to send the plant information as json
// encoded data.
type JsonPlant struct {
	Id           int            `json:"id"`
	CommonName   string         `json:"common_name"`
	GenericName  string         `json:"generic_name"`
	SpecificName string         `json:"specific_name"`
	Logs         []JsonPlantLog `json:"logs"`
}

// jsonPlantLog describes a plant log as a json object.
type JsonPlantLog struct {
	Id        int    `json:"id"`
	PlantId   int    `json:"plant_id"`
	Desc      string `json:"desc"`
	EventType int    `json:"event_type"`
}
