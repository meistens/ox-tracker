package commands

import (
	"fmt"
	"mtracker/internal/db"
	"mtracker/internal/models"
	"mtracker/internal/service"
	"strings"
)

// CommandHandler implements service.MediaTracker
type CommandHandler struct {
	mediaRepo     *db.MediaRepository
	userMediaRepo *db.UserMediaRepository
	userRepo      *db.UserRepository
	apiClient     *service.APIClient
	mediaService  *service.MediaService
}

func NewCommandHandler(mediaRepo *db.MediaRepository, userMediaRepo *db.UserMediaRepository, userRepo *db.UserRepository, apiClient *service.APIClient, mediaService *service.MediaService) *CommandHandler {
	return &CommandHandler{
		mediaRepo:     mediaRepo,
		userMediaRepo: userMediaRepo,
		userRepo:      userRepo,
		apiClient:     apiClient,
		mediaService:  mediaService,
	}
}

func (h *CommandHandler) HandleBotCommand(cmd *models.BotCommand) *models.BotResponse {
	switch cmd.Command {
	case "search":
		return h.handleSearch(cmd)
	case "list":
		return h.handleList(cmd)
	case "add":
		return h.handleAdd(cmd)
	default:
		return &models.BotResponse{
			Message: "Unknown command. Type /help for available commands.",
			Success: false,
		}
	}
}

func (h *CommandHandler) handleSearch(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /search <type> <query>\nExample: /search movie foo",
			Success: false,
		}
	}

	mediaType := cmd.Args[0]
	query := strings.Join(cmd.Args[1:], " ")

	// First, search in database
	results, err := h.mediaRepo.SearchMedia(mediaType, query, 5)
	if err != nil {
		return &models.BotResponse{
			Message: "Error searching database: " + err.Error(),
			Success: false,
		}
	}

	// If no results in database, try external API
	if len(results) == 0 {
		externalResults, err := h.searchExternalAPI(mediaType, query)
		if err != nil {
			return &models.BotResponse{
				Message: fmt.Sprintf("No %s found matching '%s'", mediaType, query),
				Success: true,
			}
		}
		results = externalResults
	}

	// Format results
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Search results for %s '%s':\n\n", mediaType, query))

	for i, media := range results {
		response.WriteString(fmt.Sprintf("%d. %s\n", i+1, media.Title))
		response.WriteString(fmt.Sprintf("   Rating: %.1f/10\n", media.Rating))
		response.WriteString(fmt.Sprintf("   Year: %s\n", media.ReleaseDate))
		response.WriteString(fmt.Sprintf("   ID: %d\n\n", media.ID))
	}

	return &models.BotResponse{
		Message: response.String(),
		Success: true,
	}
}

func (h *CommandHandler) searchExternalAPI(mediaType, query string) ([]models.Media, error) {
	switch mediaType {
	case "anime":
		return h.searchAnime(query)
	default:
		return nil, fmt.Errorf("external API not available for type: %s", mediaType)
	}
}

func (h *CommandHandler) searchAnime(query string) ([]models.Media, error) {
	// Search using Jikan API
	animeResults, err := h.apiClient.SearchAnime(query)
	if err != nil {
		return nil, err
	}

	// Convert Jikan results to Media models and save to database
	var mediaResults []models.Media
	for _, anime := range animeResults {
		// Create Media model from Jikan result
		media := models.Media{
			ExternalID:  fmt.Sprintf("mal_%d", anime.MalID),
			Title:       anime.Title,
			Type:        models.MediaTypeAnime,
			Description: anime.Synopsis,
			ReleaseDate: anime.Aired.From,
			PosterURL:   anime.Images.JPG.ImageURL,
			Rating:      anime.Score,
		}

		// Save to database
		inserted, err := h.mediaRepo.CreateMedia(&media)
		if err != nil {
			continue // Skip if error, but continue with other results
		}

		if inserted {
			mediaResults = append(mediaResults, media)
		} else {
			// If not inserted (already exists), get the existing record
			existing, err := h.mediaRepo.GetByExtID(media.ExternalID)
			if err == nil {
				mediaResults = append(mediaResults, *existing)
			}
		}

		// Limit results
		if len(mediaResults) >= 5 {
			break
		}
	}

	return mediaResults, nil
}

func (h *CommandHandler) handleList(cmd *models.BotCommand) *models.BotResponse {
	// Get user's media list
	userMedia, err := h.userMediaRepo.GetByUser(cmd.UserID, "")
	if err != nil {
		return &models.BotResponse{
			Message: "Error fetching your list: " + err.Error(),
			Success: false,
		}
	}

	if len(userMedia) == 0 {
		return &models.BotResponse{
			Message: "Your list is empty! Use /search to find media to add",
			Success: true,
		}
	}

	// Format user's media list
	var response strings.Builder
	response.WriteString("Your Media List:\n\n")

	for i, um := range userMedia {
		// Get media details
		media, err := h.mediaRepo.GetByID(um.MediaID)
		if err != nil {
			continue // Skip if media not found
		}

		response.WriteString(fmt.Sprintf("%d. %s\n", i+1, media.Title))
		response.WriteString(fmt.Sprintf("   ID: %d\n", um.MediaID))
		response.WriteString(fmt.Sprintf("   Status: %s\n", um.Status))
		response.WriteString(fmt.Sprintf("   Progress: %d\n", um.Progress))
		if um.Rating > 0 {
			response.WriteString(fmt.Sprintf("   Rating: %.1f/10\n", um.Rating))
		}
		response.WriteString("\n")
	}

	return &models.BotResponse{
		Message: response.String(),
		Success: true,
	}
}

func (h *CommandHandler) handleAdd(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 1 {
		return &models.BotResponse{
			Message: "Usage: /add <media_id or media_name>\nExamples: /add 1 or /add shawshank",
			Success: false,
		}
	}

	// First, ensure user exists
	user := &models.User{
		ID:       cmd.UserID,
		Username: "user", // Default username
		Platform: "telegram",
	}
	err := h.userRepo.CreateUser(user)
	if err != nil {
		return &models.BotResponse{
			Message: "Error creating user: " + err.Error(),
			Success: false,
		}
	}

	// Try to parse as ID first
	var mediaID int
	var media *models.Media

	if _, err := fmt.Sscanf(cmd.Args[0], "%d", &mediaID); err == nil {
		// It's a numeric ID
		media, err = h.mediaRepo.GetByID(mediaID)
		if err != nil {
			return &models.BotResponse{
				Message: "Media not found with that ID. Use /search to find valid media IDs.",
				Success: false,
			}
		}

		// Use service method to add media to user
		addedMedia, err := h.mediaService.AddMediaToUser(cmd.UserID, media.ExternalID, media.Title, media.Type)
		if err != nil {
			return &models.BotResponse{
				Message: "Error adding media to your list: " + err.Error(),
				Success: false,
			}
		}

		return &models.BotResponse{
			Message: fmt.Sprintf("Added '%s' to your watchlist!", addedMedia.Title),
			Success: true,
		}
	} else {
		// It's a name, search for it across all types
		query := strings.Join(cmd.Args, " ")

		// Try searching in different media types to find a match
		mediaTypes := []string{"movie", "tv", "anime", "book"}
		var bestMatch *models.Media

		for _, mediaType := range mediaTypes {
			results, err := h.mediaRepo.SearchMedia(mediaType, query, 1)
			if err != nil {
				continue
			}

			if len(results) > 0 {
				bestMatch = &results[0]
				break
			}
		}

		if bestMatch == nil {
			return &models.BotResponse{
				Message: "No media found with that name. Use /search to find media first.",
				Success: false,
			}
		}

		// Use service method to add media to user
		addedMedia, err := h.mediaService.AddMediaToUser(cmd.UserID, bestMatch.ExternalID, bestMatch.Title, bestMatch.Type)
		if err != nil {
			return &models.BotResponse{
				Message: "Error adding media to your list: " + err.Error(),
				Success: false,
			}
		}

		return &models.BotResponse{
			Message: fmt.Sprintf("Added '%s' to your watchlist!", addedMedia.Title),
			Success: true,
		}
	}
}

// Ensure CommandHandler implements the interface
var _ service.MediaTracker = (*CommandHandler)(nil)
