package service

import (
	"encoding/json"
	"fmt"
	"mtracker/internal/db"
	"mtracker/internal/models"
	"net/http"
	"time"
)

// Circular import prevention
type MediaTracker interface {
	HandleBotCommand(cmd *models.BotCommand) *models.BotResponse
}

type APIClient struct {
	tmdbAPIKey string
	httpClient *http.Client
}

func NewAPIClient(tmdbAPIKey string) *APIClient {
	return &APIClient{
		tmdbAPIKey: tmdbAPIKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// TBA when I can get a domain up and running or get a replacement
func (t *APIClient) SearchTMDB(query string, mediaType models.MediaType) ([]models.TMDBMedia, error) {
	var endpoint string

	switch mediaType {
	case models.MediaTypeMovie:
		endpoint = "movie"
	case models.MediaTypeTV:
		endpoint = "tv"
	default:
		return nil, fmt.Errorf("unsupported media type for TMDB: %s", mediaType)
	}

	url := fmt.Sprintf("https://api.themoviedb.org/3/search/%s?api_key=%s&query=%s",
		endpoint, t.tmdbAPIKey, query)

	resp, err := t.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp models.TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Results, nil
}

// Jikan API SearchAnime
func (t *APIClient) SearchAnime(query string) ([]models.JikanAnime, error) {
	url := fmt.Sprintf("https://api.jikan.moe/v4/anime?q=%s&limit=10", query)

	resp, err := t.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp models.JikanSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Data, nil
}

// MedisService handles media-related logic
type MediaService struct {
	repositories *db.Repositories
	apiClient    *APIClient
}

func NewMediaService(repos *db.Repositories, apiClient *APIClient) *MediaService {
	return &MediaService{
		repositories: repos,
		apiClient:    apiClient,
	}
}

// TODO: add TMDB/replacement and find an OpenLibrary alternative
func (s *MediaService) SearchMedia(query string, mediaType models.MediaType) (interface{}, error) {
	switch mediaType {
	case models.MediaTypeAnime:
		return s.apiClient.SearchAnime(query)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}
