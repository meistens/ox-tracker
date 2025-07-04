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
	case "status":
		return h.handleStatus(cmd)
	case "rate":
		return h.handleRate(cmd)
	case "progress":
		return h.handleProgress(cmd)
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

func (h *CommandHandler) handleStatus(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /status <media_id> <status>\nExample: /status 1 completed\nAvailable statuses: watching, completed, plan_to_read, on_hold, dropped, watchlist",
			Success: false,
		}
	}

	// Parse media ID
	var mediaID int
	if _, err := fmt.Sscanf(cmd.Args[0], "%d", &mediaID); err != nil {
		return &models.BotResponse{
			Message: "Invalid media ID. Please provide a numeric ID.",
			Success: false,
		}
	}

	// Parse status
	statusStr := strings.ToLower(cmd.Args[1])
	var status models.Status
	switch statusStr {
	case "watching":
		status = models.StatusWatching
	case "completed":
		status = models.StatusCompleted
	case "plan_to_read":
		status = models.StatusPlanToRead
	case "on_hold":
		status = models.StatusOnHold
	case "dropped":
		status = models.StatusDropped
	case "watchlist":
		status = models.StatusWatchlist
	default:
		return &models.BotResponse{
			Message: "Invalid status. Available statuses: watching, completed, plan_to_read, on_hold, dropped, watchlist",
			Success: false,
		}
	}

	// Ensure user exists
	user := &models.User{
		ID:       cmd.UserID,
		Username: "user",
		Platform: "telegram",
	}
	err := h.userRepo.CreateUser(user)
	if err != nil {
		return &models.BotResponse{
			Message: "Error creating user: " + err.Error(),
			Success: false,
		}
	}

	// Check if media exists
	media, err := h.mediaRepo.GetByID(mediaID)
	if err != nil {
		return &models.BotResponse{
			Message: "Media not found with that ID. Use /search to find valid media IDs.",
			Success: false,
		}
	}

	// Update status using service method
	err = h.mediaService.UpdateUserMediaStatus(cmd.UserID, mediaID, status)
	if err != nil {
		return &models.BotResponse{
			Message: "Error updating status: " + err.Error(),
			Success: false,
		}
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("Updated status for '%s' to %s!", media.Title, statusStr),
		Success: true,
	}
}

func (h *CommandHandler) handleRate(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /rate <media_id> <rating>\nExample: /rate 1 8.5\nRating should be between 0.0 and 10.0",
			Success: false,
		}
	}

	// Parse media ID
	var mediaID int
	if _, err := fmt.Sscanf(cmd.Args[0], "%d", &mediaID); err != nil {
		return &models.BotResponse{
			Message: "Invalid media ID. Please provide a numeric ID.",
			Success: false,
		}
	}

	// Parse rating
	var rating float64
	if _, err := fmt.Sscanf(cmd.Args[1], "%f", &rating); err != nil {
		return &models.BotResponse{
			Message: "Invalid rating. Please provide a number between 0.0 and 10.0.",
			Success: false,
		}
	}

	// Validate rating range
	if rating < 0.0 || rating > 10.0 {
		return &models.BotResponse{
			Message: "Rating must be between 0.0 and 10.0.",
			Success: false,
		}
	}

	// Ensure user exists
	user := &models.User{
		ID:       cmd.UserID,
		Username: "user",
		Platform: "telegram",
	}
	err := h.userRepo.CreateUser(user)
	if err != nil {
		return &models.BotResponse{
			Message: "Error creating user: " + err.Error(),
			Success: false,
		}
	}

	// Check if media exists
	media, err := h.mediaRepo.GetByID(mediaID)
	if err != nil {
		return &models.BotResponse{
			Message: "Media not found with that ID. Use /search to find valid media IDs.",
			Success: false,
		}
	}

	// Rate media using service method
	err = h.mediaService.RateMedia(cmd.UserID, mediaID, rating)
	if err != nil {
		return &models.BotResponse{
			Message: "Error rating media: " + err.Error(),
			Success: false,
		}
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("Rated '%s' with %.1f/10 stars!", media.Title, rating),
		Success: true,
	}
}

func (h *CommandHandler) handleProgress(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /progress <media_id> <episode_number>\nExample: /progress 1 5\nUse 0 to reset progress",
			Success: false,
		}
	}

	// Parse media ID
	var mediaID int
	if _, err := fmt.Sscanf(cmd.Args[0], "%d", &mediaID); err != nil {
		return &models.BotResponse{
			Message: "Invalid media ID. Please provide a numeric ID.",
			Success: false,
		}
	}

	// Parse progress
	var progress int
	if _, err := fmt.Sscanf(cmd.Args[1], "%d", &progress); err != nil {
		return &models.BotResponse{
			Message: "Invalid progress. Please provide a number (episode number).",
			Success: false,
		}
	}

	// Validate progress range
	if progress < 0 {
		return &models.BotResponse{
			Message: "Progress cannot be negative. Use 0 to reset progress.",
			Success: false,
		}
	}

	// Ensure user exists
	user := &models.User{
		ID:       cmd.UserID,
		Username: "user",
		Platform: "telegram",
	}
	err := h.userRepo.CreateUser(user)
	if err != nil {
		return &models.BotResponse{
			Message: "Error creating user: " + err.Error(),
			Success: false,
		}
	}

	// Check if media exists
	media, err := h.mediaRepo.GetByID(mediaID)
	if err != nil {
		return &models.BotResponse{
			Message: "Media not found with that ID. Use /search to find valid media IDs.",
			Success: false,
		}
	}

	// Update progress using service method
	err = h.mediaService.UpdateProgress(cmd.UserID, mediaID, progress)
	if err != nil {
		return &models.BotResponse{
			Message: "Error updating progress: " + err.Error(),
			Success: false,
		}
	}

	// Determine status message based on progress
	var statusMsg string
	if progress == 0 {
		statusMsg = "Reset progress"
	} else {
		statusMsg = fmt.Sprintf("Updated progress to episode %d", progress)
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("%s for '%s'!", statusMsg, media.Title),
		Success: true,
	}
}

// Ensure CommandHandler implements the interface
var _ service.MediaTracker = (*CommandHandler)(nil)
