package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mtracker/internal/models"
	"mtracker/internal/service"
	"net/http"
	"strconv"
	"strings"
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

// Process incoming webhook updates
func (t *TelegramHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Failed to unmarshal update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	t.handleUpdate(update)
	w.WriteHeader(http.StatusOK)
}

func (t *TelegramHandler) handleUpdate(update Update) {
	message := update.Message

	// ignore msg from bots
	if message.From.IsBot {
		return
	}

	//ignore empty msgs
	if message.Text == "" {
		return
	}

	// handle commands
	// TODO: handleCommand function
	if strings.HasPrefix(message.Text, t.prefix) {
		t.handleCommand(message)
		return
	}

	// handle plaintext
	// TODO: handlePlaintext function
	if message.Chat.Type == "private" {
		t.handlePlaintext(message)
	}
}

func (t *TelegramHandler) handleCommand(message Message) {
	text := strings.TrimPrefix(message.Text, t.prefix)
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	// handle help and start commands locally
	if command == "help" || command == "start" {
		// TODO: sendHelpMessage
		t.sendHelpMessage(message.Chat.ID)
		return
	}

	// create bot command
	botCmd := &models.BotCommand{
		Command: command,
		Args:    args,
		UserID:  strconv.Itoa(message.From.ID),
	}

	// handle command through media tracker
	response := t.mediaTracker.HandleBotCommand(botCmd)

	// send response
	// TODO: sendResponse
	t.sendResponse(message.Chat.ID, response, command)
}

// handlePlaintext
func (t *TelegramHandler) handlePlaintext(message Message) {
	// TODO: inline search in place of cmd suggestions
	text := "use commands to interact with the bt\n\nType /help to see available commands"
	// TODO: sendMessage
	t.sendMessage(message.Chat.ID, text, "Markdown")
}

// sendResponse
func (t *TelegramHandler) sendResponse(chatID int64, response *models.BotResponse, command string) {
	var emoji string
	var text strings.Builder

	if response.Success {
		// getCommandTitle
		text.WriteString(fmt.Sprintf("%s *%s*\n\n", emoji, t.getCommandTitle(command)))
	} else {
		emoji = "⏰"
		text.WriteString(fmt.Sprintf("%s *Error*\n\n", emoji))
	}

	// format response message
	// TODO: formatResponseMessage
	responseText := t.formatResponseText(response.Message, command)
	text.WriteString(responseText)

	t.sendMessage(chatID, text.String(), "Markdown")
}

// formatResponseText
func (t *TelegramHandler) formatResponseText(message, command string) string {
	// TODO: formatSearchResults
	switch command {
	case "search":
		return t.formatSearchResults(message)
	case "list":
		return t.formatListResults(message)
	default:
		return message
	}
}

// formatSearchResults
func (t *TelegramHandler) formatSearchResults(message string) string {
	// Replace numbers with emojis of choice and add formatting, or get rid of it...
	formatted := message
	formatted = strings.ReplaceAll(formatted, "1.", "1️⃣")
	formatted = strings.ReplaceAll(formatted, "2.", "2️⃣")
	formatted = strings.ReplaceAll(formatted, "3.", "3️⃣")
	formatted = strings.ReplaceAll(formatted, "4.", "4️⃣")
	formatted = strings.ReplaceAll(formatted, "5.", "5️⃣")

	// ID formatting
	lines := strings.Split(formatted, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Rating:") {
			lines[i] = strings.ReplaceAll(line, "Rating:", "5️⃣  *Rating:*")
		}
	}
	return strings.Join(lines, "\n")
}

// formatListResults
func (t *TelegramHandler) formatListResults(message string) string {
	if strings.Contains(message, "No media found") {
		return "Your list is empty! Use /search to find media to add"
	}

	formatted := message
	formatted = strings.ReplaceAll(formatted, "ID:", "🆔 *ID:*")
	formatted = strings.ReplaceAll(formatted, "Status:", "📊 *Status:*")
	formatted = strings.ReplaceAll(formatted, "Progress:", "📈 *Progress:*")
	formatted = strings.ReplaceAll(formatted, "Rating:", "⭐ *Rating:*")

	return formatted
}

// getCommandEmoji
func (t *TelegramHandler) getCommandEmoji(command string) string {
	switch command {
	case "search":
		return "🔍"
	case "add":
		return "➕"
	case "status":
		return "📝"
	case "list":
		return "📋"
	case "rate":
		return "⭐"
	case "progress":
		return "📈"
	case "remind":
		return "⏰"
	default:
		return "✅"
	}
}

// getCommandTitle
func (t *TelegramHandler) getCommandTitle(command string) string {
	switch command {
	case "search":
		return "Search Results"
	case "add":
		return "Media Added"
	case "status":
		return "Status Updated"
	case "list":
		return "Your Media List"
	case "rate":
		return "Rating Updated"
	case "progress":
		return "Progress Updated"
	case "remind":
		return "Reminder Set"
	default:
		return "Success"
	}
}
