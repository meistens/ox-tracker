package telegram

import (
	"fmt"
	"log"
	"mtracker/internal/service"
	"net/http"
	"time"
)

// TG API Types
type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`
}

type User struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type SendMessageRequest struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"` //???
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

type TelegramHandler struct {
	token        string
	mediaTracker service.MediaTracker
	httpClient   *http.Client
	baseURL      string
	prefix       string
}

func NewTelegramHandler(token string, mediaTracker service.MediaTracker) *TelegramHandler {
	return &TelegramHandler{
		token:        token,
		mediaTracker: mediaTracker,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      fmt.Sprintf("https://api.telegram.org/bot%s", token),
		prefix:       "/",
	}
}

func (t *TelegramHandler) Start() error {
	// TODO: set webhook or start polling
	log.Println("Telegram bot is now running")
	return nil
}

func (t *TelegramHandler) Stop() error {
	log.Println("Telegram bot stopped")
	return nil
}
