package commands

import (
	"fmt"
	"mtracker/internal/db"
	"mtracker/internal/models"
	"mtracker/internal/service"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	case "getlist":
		return h.handleGetList(cmd)
	case "remind":
		return h.handleRemind(cmd)
	case "delete":
		return h.handleDelete(cmd)
	case "notes":
		return h.handleNotes(cmd)
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

		// Display progress based on the new format
		if um.Progress.Current > 0 {
			if um.Progress.Total > 0 {
				response.WriteString(fmt.Sprintf("   Progress: %s (%s)\n", um.Progress.Details, um.Progress.Unit))
			} else {
				response.WriteString(fmt.Sprintf("   Progress: %s %s\n", um.Progress.Details, um.Progress.Unit))
			}
		} else if um.Progress.Details == "completed" {
			response.WriteString("   Progress: Completed\n")
		}

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

func (h *CommandHandler) handleGetList(cmd *models.BotCommand) *models.BotResponse {
	// Parse optional status filter
	var status models.Status
	if len(cmd.Args) > 0 {
		statusStr := strings.ToLower(cmd.Args[0])
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
		case "all":
			status = ""
		default:
			return &models.BotResponse{
				Message: "Usage: /getlist [status]\nAvailable statuses: watching, completed, plan_to_read, on_hold, dropped, watchlist, all\nExample: /getlist completed",
				Success: false,
			}
		}
	}

	// Get detailed user media list using service method
	detailedList, err := h.mediaService.GetUserMediaList(cmd.UserID, status)
	if err != nil {
		return &models.BotResponse{
			Message: "Error fetching your list: " + err.Error(),
			Success: false,
		}
	}

	if len(detailedList) == 0 {
		statusMsg := "all media"
		if status != "" {
			statusMsg = string(status)
		}
		return &models.BotResponse{
			Message: fmt.Sprintf("Your %s list is empty! Use /search to find media to add", statusMsg),
			Success: true,
		}
	}

	// Format detailed user's media list
	var response strings.Builder
	statusMsg := "All Media"
	if status != "" {
		statusMsg = fmt.Sprintf("%s Media", strings.Title(string(status)))
	}
	response.WriteString(fmt.Sprintf("Your %s List:\n\n", statusMsg))

	for i, item := range detailedList {
		response.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, item.Media.Title, item.Media.Type))
		response.WriteString(fmt.Sprintf("   ID: %d\n", item.MediaID))
		response.WriteString(fmt.Sprintf("   Status: %s\n", item.Status))

		// Display progress based on the new format
		if item.Progress.Current > 0 {
			if item.Progress.Total > 0 {
				response.WriteString(fmt.Sprintf("   Progress: %s (%s)\n", item.Progress.Details, item.Progress.Unit))
			} else {
				response.WriteString(fmt.Sprintf("   Progress: %s %s\n", item.Progress.Details, item.Progress.Unit))
			}
		} else if item.Progress.Details == "completed" {
			response.WriteString("   Progress: Completed\n")
		}

		if item.Rating > 0 {
			response.WriteString(fmt.Sprintf("   Rating: %.1f/10\n", item.Rating))
		}

		if item.Notes != "" {
			response.WriteString(fmt.Sprintf("   Notes: %s\n", item.Notes))
		}

		response.WriteString("\n")
	}

	return &models.BotResponse{
		Message: response.String(),
		Success: true,
	}
}

func (h *CommandHandler) handleRemind(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) == 0 {
		// List reminders
		return h.listReminders(cmd)
	}

	if len(cmd.Args) < 3 {
		return &models.BotResponse{
			Message: "Usage: /remind <media_id> <time> <message>\nExamples:\n  /remind 1 2h Continue watching\n  /remind 1 1d Watch next episode\n  /remind 1 30m Take a break\n  /remind (to list your reminders)",
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

	// Parse time duration
	durationStr := cmd.Args[1]
	remindAt, err := h.parseReminderTime(durationStr)
	if err != nil {
		return &models.BotResponse{
			Message: "Invalid time format. Examples: 30m, 2h, 1d, 1w",
			Success: false,
		}
	}

	// Get message
	message := strings.Join(cmd.Args[2:], " ")
	if message == "" {
		message = "Time to continue watching!"
	}

	// Ensure user exists
	user := &models.User{
		ID:       cmd.UserID,
		Username: "user",
		Platform: "telegram",
	}
	err = h.userRepo.CreateUser(user)
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

	// Create reminder using service method
	reminder, err := h.mediaService.CreateReminder(cmd.UserID, mediaID, message, remindAt)
	if err != nil {
		return &models.BotResponse{
			Message: "Error creating reminder: " + err.Error(),
			Success: false,
		}
	}

	// Format reminder time for display
	timeUntil := time.Until(remindAt)
	var timeStr string
	if timeUntil.Hours() >= 24 {
		days := int(timeUntil.Hours() / 24)
		timeStr = fmt.Sprintf("%d day(s)", days)
	} else if timeUntil.Hours() >= 1 {
		hours := int(timeUntil.Hours())
		timeStr = fmt.Sprintf("%d hour(s)", hours)
	} else {
		minutes := int(timeUntil.Minutes())
		timeStr = fmt.Sprintf("%d minute(s)", minutes)
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("Reminder set for '%s' in %s!\nMessage: %s", media.Title, timeStr, reminder.Message),
		Success: true,
	}
}

func (h *CommandHandler) listReminders(cmd *models.BotCommand) *models.BotResponse {
	// Get user's reminders
	reminders, err := h.mediaService.GetUserReminders(cmd.UserID)
	if err != nil {
		return &models.BotResponse{
			Message: "Error fetching reminders: " + err.Error(),
			Success: false,
		}
	}

	if len(reminders) == 0 {
		return &models.BotResponse{
			Message: "You have no reminders set.",
			Success: true,
		}
	}

	// Format reminders list
	var response strings.Builder
	response.WriteString("Your Reminders:\n\n")

	for i, reminder := range reminders {
		// Get media details
		media, err := h.mediaRepo.GetByID(reminder.MediaID)
		if err != nil {
			continue // Skip if media not found
		}

		response.WriteString(fmt.Sprintf("%d. %s\n", i+1, media.Title))
		response.WriteString(fmt.Sprintf("   Message: %s\n", reminder.Message))

		// Format reminder time
		if reminder.Sent {
			response.WriteString("   Status: Sent\n")
		} else {
			timeUntil := time.Until(reminder.RemindAt)
			if timeUntil > 0 {
				if timeUntil.Hours() >= 24 {
					days := int(timeUntil.Hours() / 24)
					response.WriteString(fmt.Sprintf("   Reminds in: %d day(s)\n", days))
				} else if timeUntil.Hours() >= 1 {
					hours := int(timeUntil.Hours())
					response.WriteString(fmt.Sprintf("   Reminds in: %d hour(s)\n", hours))
				} else {
					minutes := int(timeUntil.Minutes())
					response.WriteString(fmt.Sprintf("   Reminds in: %d minute(s)\n", minutes))
				}
			} else {
				response.WriteString("   Status: Overdue\n")
			}
		}
		response.WriteString("\n")
	}

	return &models.BotResponse{
		Message: response.String(),
		Success: true,
	}
}

func (h *CommandHandler) handleDelete(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 1 {
		return &models.BotResponse{
			Message: "Usage: /delete <media_id>\nExample: /delete 1",
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

	// Delete media from user's list using service method
	media, err := h.mediaService.DeleteMediaFromUser(cmd.UserID, mediaID)
	if err != nil {
		return &models.BotResponse{
			Message: "Error removing media from your list: " + err.Error(),
			Success: false,
		}
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("Removed '%s' from your list!", media.Title),
		Success: true,
	}
}

func (h *CommandHandler) handleNotes(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /notes <media_id> <note_text>\nExamples:\n  /notes 1 Great series, highly recommend!\n  /notes 1 Watch with friends\n  /notes 1 (to clear notes)",
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

	// Get note text (all remaining arguments)
	noteText := strings.Join(cmd.Args[1:], " ")

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

	// Update notes using service method
	_, err = h.mediaService.UpdateUserMediaNotes(cmd.UserID, mediaID, noteText)
	if err != nil {
		return &models.BotResponse{
			Message: "Error updating notes: " + err.Error(),
			Success: false,
		}
	}

	// Create response message
	var responseMsg string
	if noteText == "" {
		responseMsg = fmt.Sprintf("Cleared notes for '%s'", media.Title)
	} else {
		responseMsg = fmt.Sprintf("Updated notes for '%s': %s", media.Title, noteText)
	}

	return &models.BotResponse{
		Message: responseMsg,
		Success: true,
	}
}

// parseProgress parses different progress formats and returns a Progress struct
func parseProgress(input string, mediaType models.MediaType) (*models.Progress, error) {
	input = strings.TrimSpace(input)

	// Handle percentage format: "50%"
	if strings.HasSuffix(input, "%") {
		percentStr := strings.TrimSuffix(input, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage format")
		}
		if percent < 0 || percent > 100 {
			return nil, fmt.Errorf("percentage must be between 0 and 100")
		}
		return &models.Progress{
			Current: percent,
			Total:   100,
			Unit:    "percentage",
			Details: fmt.Sprintf("%.1f%%", percent),
		}, nil
	}

	// Handle fraction format: "5/12" or "150/300"
	if strings.Contains(input, "/") {
		parts := strings.Split(input, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid fraction format, use 'current/total'")
		}

		current, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid current value in fraction")
		}

		total, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid total value in fraction")
		}

		if current < 0 || total <= 0 || current > total {
			return nil, fmt.Errorf("invalid fraction: current must be 0-%v, total must be positive", total)
		}

		unit := getUnitForMediaType(mediaType)
		return &models.Progress{
			Current: current,
			Total:   total,
			Unit:    unit,
			Details: fmt.Sprintf("%.0f/%.0f", current, total),
		}, nil
	}

	// Handle season-episode format: "s2e5" or "S2E5"
	seasonEpisodeRegex := regexp.MustCompile(`(?i)^s(\d+)e(\d+)$`)
	if match := seasonEpisodeRegex.FindStringSubmatch(input); match != nil {
		season, _ := strconv.ParseFloat(match[1], 64)
		episode, _ := strconv.ParseFloat(match[2], 64)

		return &models.Progress{
			Current: episode,
			Total:   0, // Unknown total
			Unit:    "episodes",
			Details: fmt.Sprintf("S%.0fE%.0f", season, episode),
		}, nil
	}

	// Handle simple number (episode/chapter number)
	if num, err := strconv.ParseFloat(input, 64); err == nil {
		if num < 0 {
			return nil, fmt.Errorf("progress cannot be negative")
		}

		unit := getUnitForMediaType(mediaType)
		return &models.Progress{
			Current: num,
			Total:   0, // Unknown total
			Unit:    unit,
			Details: fmt.Sprintf("%.0f", num),
		}, nil
	}

	// Handle special keywords
	switch strings.ToLower(input) {
	case "watched", "completed":
		return &models.Progress{
			Current: 1,
			Total:   1,
			Unit:    "watched",
			Details: "completed",
		}, nil
	case "unwatched", "reset":
		return &models.Progress{
			Current: 0,
			Total:   0,
			Unit:    "episodes",
			Details: "reset",
		}, nil
	}

	return nil, fmt.Errorf("invalid progress format. Examples: '5/12', 's2e5', '50%', '5', 'watched'")
}

// getUnitForMediaType returns the appropriate unit for a media type
func getUnitForMediaType(mediaType models.MediaType) string {
	switch mediaType {
	case models.MediaTypeMovie:
		return "watched"
	case models.MediaTypeTV, models.MediaTypeAnime:
		return "episodes"
	case models.MediaTypeBook:
		return "chapters"
	default:
		return "episodes"
	}
}

func (h *CommandHandler) handleProgress(cmd *models.BotCommand) *models.BotResponse {
	if len(cmd.Args) < 2 {
		return &models.BotResponse{
			Message: "Usage: /progress <media_id> <progress>\nExamples:\n  /progress 1 5/12 (episode 5 of 12)\n  /progress 1 s2e5 (season 2 episode 5)\n  /progress 1 50% (50% complete)\n  /progress 1 watched (mark as watched)\n  /progress 1 5 (episode 5)",
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

	// Parse progress input
	progressInput := strings.Join(cmd.Args[1:], " ")
	progress, err := parseProgress(progressInput, media.Type)
	if err != nil {
		return &models.BotResponse{
			Message: "Error parsing progress: " + err.Error(),
			Success: false,
		}
	}

	// Update progress using service method
	err = h.mediaService.UpdateProgress(cmd.UserID, mediaID, *progress)
	if err != nil {
		return &models.BotResponse{
			Message: "Error updating progress: " + err.Error(),
			Success: false,
		}
	}

	// Create success message
	var statusMsg string
	switch progress.Details {
	case "completed":
		statusMsg = "Marked as watched"
	case "reset":
		statusMsg = "Reset progress"
	default:
		if progress.Total > 0 {
			statusMsg = fmt.Sprintf("Updated progress to %s", progress.Details)
		} else {
			statusMsg = fmt.Sprintf("Updated progress to %s %s", progress.Details, progress.Unit)
		}
	}

	return &models.BotResponse{
		Message: fmt.Sprintf("%s for '%s'!", statusMsg, media.Title),
		Success: true,
	}
}

// parseReminderTime parses time duration strings like "30m", "2h", "1d", "1w"
func (h *CommandHandler) parseReminderTime(durationStr string) (time.Time, error) {
	// Remove any whitespace
	durationStr = strings.TrimSpace(durationStr)

	// Parse duration with unit
	var duration time.Duration

	switch {
	case strings.HasSuffix(durationStr, "m"):
		minutes, err := strconv.Atoi(strings.TrimSuffix(durationStr, "m"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid minutes format")
		}
		duration = time.Duration(minutes) * time.Minute

	case strings.HasSuffix(durationStr, "h"):
		hours, err := strconv.Atoi(strings.TrimSuffix(durationStr, "h"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid hours format")
		}
		duration = time.Duration(hours) * time.Hour

	case strings.HasSuffix(durationStr, "d"):
		days, err := strconv.Atoi(strings.TrimSuffix(durationStr, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid days format")
		}
		duration = time.Duration(days) * 24 * time.Hour

	case strings.HasSuffix(durationStr, "w"):
		weeks, err := strconv.Atoi(strings.TrimSuffix(durationStr, "w"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid weeks format")
		}
		duration = time.Duration(weeks) * 7 * 24 * time.Hour

	default:
		return time.Time{}, fmt.Errorf("invalid time format, use: 30m, 2h, 1d, 1w")
	}

	if duration <= 0 {
		return time.Time{}, fmt.Errorf("duration must be positive")
	}

	return time.Now().Add(duration), nil
}

// Ensure CommandHandler implements the interface
var _ service.MediaTracker = (*CommandHandler)(nil)
