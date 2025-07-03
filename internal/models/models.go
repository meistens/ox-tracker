package models

import "time"

// Media types|functionalities needed
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
	MediaTypeBook  MediaType = "book"
)

// Status Types|functionalities needed
type Status string

const (
	StatusWatching   Status = "watching"
	StatusCompleted  Status = "completed"
	StatusPlanToRead Status = "plan_to_read"
	StatusOnHold     Status = "on_hold"
	StatusDropped    Status = "dropped"
	StatusWatchlist  Status = "watchlist"
)

// Personalized Models, taken some ideas from
// models of API to consume
type Media struct {
	ID          int       `json:"id" db:"id"`
	ExternalID  string    `json:"external_id" db:"external_id"`
	Title       string    `json:"title" db:"title"`
	Type        MediaType `json:"type" db:"type"`
	Description string    `json:"description" db:"description"`
	ReleaseDate string    `json:"release_date" db:"release_date"`
	PosterURL   string    `json:"poster_url" db:"poster_url"`
	Rating      float64   `json:"rating" db:"rating"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Platform  string    `json:"platform" db:"platform"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UserMedia struct {
	ID        int       `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	MediaID   int       `json:"media_id" db:"media_id"`
	Status    Status    `json:"status" db:"status"`
	Progress  int       `json:"progress" db:"progress"`
	Rating    float64   `json:"rating" db:"rating"`
	Notes     string    `json:"notes" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Reminder struct {
	ID        int       `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	MediaID   int       `json:"media_id" db:"media_id"`
	Message   string    `json:"message" db:"message"`
	RemindAt  time.Time `json:"remind_at" db:"remind_at"`
	Sent      bool      `json:"sent" db:"sent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// API Response Models
// Discover section
type TMDBSearchResponse struct {
	Results []TMDBMedia `json:"results"`
}

type TMDBMedia struct {
	Adult            bool     `json:"adult"`
	GenreID          []int    `json:"genre_ids"`
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	OriginalLanguage string   `json:"original_language"`
	OriginalTitle    string   `json:"original_title"`
	Name             string   `json:"name"` // For TV shows
	Overview         string   `json:"overview"`
	OriginCountry    []string `json:"origin_country"`
	ReleaseDate      string   `json:"release_date"`
	FirstAirDate     string   `json:"first_air_date"` // For TV shows
	PosterPath       string   `json:"poster_path"`
	Popularity       float64  `json:"popularity"`
	VoteCount        int      `json:"vote_count"`

	VoteAverage float64 `json:"vote_average"`
}

type JikanSearchResponse struct {
	Data []JikanAnime `json:"data"`
}

type JikanAnime struct {
	MalID    int    `json:"mal_id"`
	Title    string `json:"title"`
	Synopsis string `json:"synopsis"`
	Aired    struct {
		From string `json:"from"`
	} `json:"aired"`
	Images struct {
		JPG struct {
			ImageURL string `json:"image_url"`
		} `json:"jpg"`
	} `json:"images"`
	Score float64 `json:"score"`
}

// Bot Command Models
// Should work for Discord as well...
type BotCommand struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	UserID  string   `json:"user_id"`
}

type BotResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Combined Models for API responses
type UserMediaWithDetails struct {
	UserMedia
	Media Media `json:"media"`
}
