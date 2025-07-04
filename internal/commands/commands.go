package commands

import (
	"fmt"
	"mtracker/internal/models"
	"mtracker/internal/service"
)

// CommandHandler implements service.MediaTracker
// Add dependencies (e.g., DB, repos) as fields if needed
// For now, it's empty

// CommandHandler is the main command logic handler
// You can add fields for DB, repos, etc. if needed
// e.g., db *db.DB

type CommandHandler struct{}

func NewCommandHandler() *CommandHandler {
	return &CommandHandler{}
}

func (h *CommandHandler) HandleBotCommand(cmd *models.BotCommand) *models.BotResponse {
	switch cmd.Command {
	case "search":
		if len(cmd.Args) < 2 {
			return &models.BotResponse{
				Message: "Usage: /search <type> <query>\nExample: /search movie foo",
				Success: false,
			}
		}
		mediaType := cmd.Args[0]
		query := cmd.Args[1]
		// TODO: Implement real search logic
		return &models.BotResponse{
			Message: fmt.Sprintf("Search results for %s '%s':\n\n1. Mock Result 1\n   Rating: 8.5/10\n   Year: 2023\n\n2. Mock Result 2\n   Rating: 7.8/10\n   Year: 2022", mediaType, query),
			Success: true,
		}
	default:
		return &models.BotResponse{
			Message: "Unknown command. Type /help for available commands.",
			Success: false,
		}
	}
}

// Ensure CommandHandler implements the interface
var _ service.MediaTracker = (*CommandHandler)(nil)
