package seed

import (
	"encoding/json"
	"log"
	"os"

	"mtracker/internal/db"
	"mtracker/internal/models"
)

func SeedMediaFromJSON(database *db.DB, jsonPath string) {
	file, err := os.Open(jsonPath)
	if err != nil {
		log.Fatalf("Failed to open seed file: %v", err)
	}
	defer file.Close()

	var mediaItems []models.Media
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&mediaItems); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}

	r := db.NewMediaRepository(database)
	for _, m := range mediaItems {
		if err := r.CreateMedia(&m); err != nil {
			log.Printf("Failed to insert media %s: %v", m.Title, err)
		} else {
			log.Printf("Inserted media: %s", m.Title)
		}
	}
}
