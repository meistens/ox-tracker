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
		inserted, err := r.CreateMedia(&m)
		if err != nil {
			log.Printf("Failed to insert media %s: %v", m.Title, err)
		} else if inserted {
			log.Printf("Inserted media: %s", m.Title)
		} else {
			log.Printf("Media %s already exists, skipping", m.Title)
		}
	}
}

func SeedUserMediaFromJSON(database *db.DB, jsonPath string) {
	file, err := os.Open(jsonPath)
	if err != nil {
		log.Fatalf("Failed to open user_media seed file: %v", err)
	}
	defer file.Close()

	var userMediaItems []models.UserMedia
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&userMediaItems); err != nil {
		log.Fatalf("Failed to decode user_media JSON: %v", err)
	}

	r := db.NewUserMediaRepository(database)
	for _, um := range userMediaItems {
		if err := r.InsertUserMedia(&um); err != nil {
			log.Printf("Failed to insert user_media for user %s and media_id %d: %v", um.UserID, um.MediaID, err)
		} else {
			log.Printf("Inserted user_media for user %s and media_id %d", um.UserID, um.MediaID)
		}
	}
}
