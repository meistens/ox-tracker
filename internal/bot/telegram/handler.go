package telegram

import (
	"bytes"
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
		httpClient:   &http.Client{Timeout: 60 * time.Second}, // Increased timeout
		baseURL:      fmt.Sprintf("https://api.telegram.org/bot%s", token),
		prefix:       "/",
	}
}

func (t *TelegramHandler) Start() error {
	log.Println("Starting Telegram bot in polling mode...")

	// Initialize offset for updates
	offset := 0

	for {
		// Get updates from Telegram API with shorter timeout
		url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=10", t.baseURL, offset)
		resp, err := t.httpClient.Get(url)
		if err != nil {
			log.Printf("Failed to get updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Telegram API error: %d", resp.StatusCode)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		var result struct {
			OK     bool     `json:"ok"`
			Result []Update `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("Failed to decode response: %v", err)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		if !result.OK {
			log.Printf("Telegram API returned error")
			time.Sleep(5 * time.Second)
			continue
		}

		// Process updates
		for _, update := range result.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			t.handleUpdate(update)
		}

		// Small delay to prevent hammering the API
		time.Sleep(1 * time.Second)
	}
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
	if strings.HasPrefix(message.Text, t.prefix) {
		t.handleCommand(message)
		return
	}

	// handle plaintext
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
	t.sendResponse(message.Chat.ID, response, command)
}

// handlePlaintext
func (t *TelegramHandler) handlePlaintext(message Message) {
	text := "use commands to interact with the bt\n\nType /help to see available commands"
	t.sendMessage(message.Chat.ID, text, "Markdown")
}

// sendResponse
func (t *TelegramHandler) sendResponse(chatID int64, response *models.BotResponse, command string) {
	var text strings.Builder

	if response.Success {
		emoji := t.getCommandEmoji(command)
		text.WriteString(fmt.Sprintf("%s *%s*\n\n", emoji, t.getCommandTitle(command)))
	} else {
		emoji := "⏰"
		text.WriteString(fmt.Sprintf("%s *Error*\n\n", emoji))
	}

	// format response message
	responseText := t.formatResponseText(response.Message, command)
	text.WriteString(responseText)

	t.sendMessage(chatID, text.String(), "Markdown")
}

// formatResponseText
func (t *TelegramHandler) formatResponseText(message, command string) string {
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

// sendHelpMessage
func (t *TelegramHandler) sendHelpMessage(chatID int64) {
	helpText := `help text here, it is a work in progress so...`

	t.sendMessage(chatID, helpText, "Markdown")
}

// sendMessage
func (t *TelegramHandler) sendMessage(chatID int64, text, parseMode string) error {
	request := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// URL for sendMessage
	// TODO: check current docs to make changes
	url := fmt.Sprintf("%s/sendMessage", t.baseURL)
	resp, err := t.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}

// All TODOs done!
//
// SendMessage implements BotIntegration interface
func (t *TelegramHandler) SendMessage(userID, message string) error {
	chatID, err := strconv.ParseInt(userID, 10, 64)

	if err != nil {
		return fmt.Errorf("invalid user ID: %v", err)
	}

	return t.sendMessage(chatID, message, "")
}

// SendReminder sends formatted reminder message
func (t *TelegramHandler) SendReminder(userID, mediaTitle, message string) error {
	chatID, err := strconv.ParseInt(userID, 10, 64)

	if err != nil {
		return fmt.Errorf("invalid user ID: %v", err)
	}

	reminderText := fmt.Sprintf("⏰ *Reminder*\n\n*%s*\n\n%s", mediaTitle, message)
	return t.sendMessage(chatID, reminderText, "Markdown")
}

// SetWebhook sets up webhook for receiving updates
func (t *TelegramHandler) SetWebhook(webhookURL string) error {
	url := fmt.Sprintf("%s/setWebhook", t.baseURL)

	request := map[string]interface{}{
		"url": webhookURL,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := t.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}

// DeleteWebhook removes webhook
func (t *TelegramHandler) DeleteWebhook() error {
	url := fmt.Sprintf("%s/deleteWebhook", t.baseURL)

	resp, err := t.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}
