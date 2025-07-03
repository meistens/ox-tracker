package service

import "mtracker/internal/models"

// Circular import prevention
type MediaTracker interface {
	HandleBotCommand(cmd *models.BotCommand) *models.BotResponse
}

type MediaTrackerImpl struct{}
