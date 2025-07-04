package service

import (
	"encoding/json"
	"fmt"
	"log"
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

func (s *MediaService) AddMediaToUser(userID, extID, title string, mediaType models.MediaType) (*models.Media, error) {
	existingMedia, err := s.repositories.Media.GetByExtID(extID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("database error: %w", err)
	}

	var media *models.Media
	if existingMedia != nil {
		media = existingMedia
	} else {
		// create new media
		media = &models.Media{
			ExternalID: extID,
			Title:      title,
			Type:       mediaType,
		}

		inserted, err := s.repositories.Media.CreateMedia(media)
		if err != nil {
			return nil, fmt.Errorf("failed to create media: %w", err)
		}
		if !inserted {
			// Media already exists, get the existing record
			existingMedia, err := s.repositories.Media.GetByExtID(extID)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing media: %w", err)
			}
			media = existingMedia
		}
	}
	// add to user's list
	userMedia := &models.UserMedia{
		UserID:  userID,
		MediaID: media.ID,
		Status:  models.StatusWatchlist,
	}

	if err := s.repositories.UserMedia.InsertUserMedia(userMedia); err != nil {
		return nil, fmt.Errorf("failed to add user list: %w", err)
	}
	return media, nil
}

func (s *MediaService) UpdateUserMediaStatus(userID string, mediaID int, status models.Status) error {
	userMedia := &models.UserMedia{
		UserID:  userID,
		MediaID: mediaID,
		Status:  status,
	}

	return s.repositories.UserMedia.InsertUserMedia(userMedia)
}

func (s *MediaService) RateMedia(userID string, mediaID int, rating float64) error {
	userMedia, err := s.repositories.UserMedia.GetByUserAndMedia(userID, mediaID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("database error: %w", err)
	}

	if userMedia == nil {
		userMedia = &models.UserMedia{
			UserID:  userID,
			MediaID: mediaID,
			Status:  models.StatusCompleted,
		}
	}

	userMedia.Rating = rating
	return s.repositories.UserMedia.InsertUserMedia(userMedia)
}

func (s *MediaService) UpdateProgress(userID string, mediaID int, progress models.Progress) error {
	userMedia, err := s.repositories.UserMedia.GetByUserAndMedia(userID, mediaID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("database error: %w", err)
	}

	if userMedia == nil {
		userMedia = &models.UserMedia{
			UserID:  userID,
			MediaID: mediaID,
			Status:  models.StatusWatching,
		}
	}

	userMedia.Progress = progress
	return s.repositories.UserMedia.InsertUserMedia(userMedia)
}

func (s *MediaService) CreateReminder(userID string, mediaID int, message string, remindAt time.Time) (*models.Reminder, error) {
	// Check if media exists
	_, err := s.repositories.Media.GetByID(mediaID)
	if err != nil {
		return nil, fmt.Errorf("media not found: %w", err)
	}

	// Create reminder
	reminder := &models.Reminder{
		UserID:   userID,
		MediaID:  mediaID,
		Message:  message,
		RemindAt: remindAt,
		Sent:     false,
	}

	err = s.repositories.Reminder.CreateReminder(reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return reminder, nil
}

func (s *MediaService) GetUserReminders(userID string) ([]models.Reminder, error) {
	return s.repositories.Reminder.GetRemindersByUser(userID)
}

func (s *MediaService) DeleteMediaFromUser(userID string, mediaID int) (*models.Media, error) {
	// Check if media exists
	media, err := s.repositories.Media.GetByID(mediaID)
	if err != nil {
		return nil, fmt.Errorf("media not found: %w", err)
	}

	// Check if user has this media in their list
	_, err = s.repositories.UserMedia.GetByUserAndMedia(userID, mediaID)
	if err != nil {
		return nil, fmt.Errorf("media not in user's list: %w", err)
	}

	// Delete from user's list
	err = s.repositories.UserMedia.Delete(userID, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete from user's list: %w", err)
	}

	return media, nil
}

func (s *MediaService) GetUserMediaList(userID string, status models.Status) ([]models.UserMediaWithDetails, error) {
	userMediaList, err := s.repositories.UserMedia.GetByUser(userID, status)
	if err != nil {
		return nil, err
	}

	var detailedList []models.UserMediaWithDetails

	for _, userMedia := range userMediaList {
		media, err := s.repositories.Media.GetByID(userMedia.MediaID)
		if err != nil {
			log.Printf("faild to get media details for ID %d: %v", userMedia.MediaID, err)
			continue
		}

		detailedList = append(detailedList, models.UserMediaWithDetails{
			UserMedia: userMedia,
			Media:     *media,
		})
	}

	return detailedList, nil
}
