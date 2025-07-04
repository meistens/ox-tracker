package db

import (
	"database/sql"
	"mtracker/internal/models"
)

// User-related database ops
type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
	INSERT INTO users (id, username, platform, updated_at)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	ON CONFLICT (id) DO UPDATE SET
	username = $2, platform = $3, updated_at = CURRENT_TIMESTAMP
	RETURNING created_at`

	err := r.db.QueryRow(query, user.ID, user.Username, user.Platform).Scan(&user.CreatedAt)
	return err
}

func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	query := `SELECT id, username, platform, created_at, updated_at
	FROM users
	WHERE id = $1`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Platform, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// Media-Related database ops
type MediaRepository struct {
	db *DB
}

func NewMediaRepository(db *DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) CreateMedia(media *models.Media) (bool, error) {
	query := `
	INSERT INTO media (external_id, title, type, description, release_date, poster_url, rating)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (external_id) DO NOTHING
	RETURNING id, created_at
	`
	err := r.db.QueryRow(query, media.ExternalID, media.Title, media.Type, media.Description, media.ReleaseDate, media.PosterURL, media.Rating).Scan(&media.ID, &media.CreatedAt)

	if err == sql.ErrNoRows {
		// TODO: extend logic to cover real usecase or modify when ready
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *MediaRepository) GetByExtID(externalID string) (*models.Media, error) {
	query := `
	SELECT id, external_id, title, type, description, release_date, poster_url, rating, created_at
	FROM media
	WHERE external_id = $1
	`

	media := &models.Media{}
	err := r.db.QueryRow(query, externalID).Scan(
		&media.ID, &media.ExternalID, &media.Title, &media.Type,
		&media.Description, &media.ReleaseDate, &media.PosterURL,
		&media.Rating, &media.CreatedAt,
	)

	if err != nil {
		return nil, err
	}
	return media, nil
}

func (r *MediaRepository) GetByID(id int) (*models.Media, error) {
	query := `
	SELECT id, external_id, title, type, description, release_date, poster_url, rating, created_at
	FROM media
	WHERE id = $1`

	media := &models.Media{}
	err := r.db.QueryRow(query, id).Scan(
		&media.ID, &media.ExternalID, &media.Title, &media.Type,
		&media.Description, &media.ReleaseDate, &media.PosterURL,
		&media.Rating, &media.CreatedAt,
	)

	if err != nil {
		return nil, err
	}
	return media, nil
}

func (r *MediaRepository) SearchMedia(mediaType string, query string, limit int) ([]models.Media, error) {
	sqlQuery := `
	SELECT id, external_id, title, type, description, release_date, poster_url, rating, created_at
	FROM media
	WHERE type = $1 AND title ILIKE $2
	ORDER BY rating DESC, title ASC
	LIMIT $3
	`

	rows, err := r.db.Query(sqlQuery, mediaType, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		err := rows.Scan(
			&media.ID, &media.ExternalID, &media.Title, &media.Type,
			&media.Description, &media.ReleaseDate, &media.PosterURL,
			&media.Rating, &media.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		mediaList = append(mediaList, media)
	}

	return mediaList, nil
}

// UserMedia handles media tracking-related ops
type UserMediaRepository struct {
	db *DB
}

func NewUserMediaRepository(db *DB) *UserMediaRepository {
	return &UserMediaRepository{db: db}
}

func (r *UserMediaRepository) InsertUserMedia(userMedia *models.UserMedia) error {
	query := `
	INSERT INTO user_media (user_id, media_id, status, progress, rating, notes, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
	ON CONFLICT (user_id, media_id)
	DO UPDATE SET status = $3, progress = $4, rating = $5, notes = $6, updated_at = CURRENT_TIMESTAMP
	RETURNING id, created_at
	`

	err := r.db.QueryRow(
		query, userMedia.UserID, userMedia.MediaID, userMedia.Status,
		userMedia.Progress, userMedia.Rating, userMedia.Notes).
		Scan(&userMedia.ID, &userMedia.CreatedAt)

	return err
}

func (r *UserMediaRepository) GetByUserAndMedia(userID string, mediaID int) (*models.UserMedia, error) {
	query := `
	SELECT id, user_id, media_id, status, progress, rating, notes, created_at, updated_at
	FROM user_media
	WHERE user_id = $1 AND media_id = $2
	`

	userMedia := &models.UserMedia{}
	err := r.db.QueryRow(query, userID, mediaID).Scan(
		&userMedia.ID, &userMedia.UserID, &userMedia.MediaID, &userMedia.Status,
		&userMedia.Progress, &userMedia.Rating, &userMedia.Notes,
		&userMedia.CreatedAt, &userMedia.UpdatedAt,
	)
	return userMedia, err

}

func (r *UserMediaRepository) GetByUser(userID string, status models.Status) ([]models.UserMedia, error) {
	query := `
	SELECT id, user_id, media_id, status, progress, rating, notes, created_at, updated_at
	FROM user_media
	WHERE user_id = $1
	`

	args := []interface{}{userID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userMediaList []models.UserMedia
	for rows.Next() {
		var newUserMedia models.UserMedia
		err := rows.Scan(&newUserMedia.ID, &newUserMedia.UserID, &newUserMedia.MediaID, &newUserMedia.Status, &newUserMedia.Progress, &newUserMedia.Rating, &newUserMedia.Notes, &newUserMedia.CreatedAt, &newUserMedia.UpdatedAt)

		if err != nil {
			return nil, err
		}
		userMediaList = append(userMediaList, newUserMedia)
	}
	return userMediaList, nil
}

func (r *UserMediaRepository) Delete(userID string, mediaID int) error {
	query := `
	DELETE FROM user_media
	WHERE user_id = $1 AND media_id = $2
	`

	_, err := r.db.Exec(query, userID, mediaID)
	return err
}

// Reminders handles reminder-related ops
type ReminderRepository struct {
	db *DB
}

func NewReminderRepository(db *DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) CreateReminder(reminder *models.Reminder) error {
	query := `
	INSERT INTO reminders (user_id, media_id, message, remind_at)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	err := r.db.QueryRow(query, reminder.UserID, reminder.MediaID,
		reminder.Message, reminder.RemindAt).
		Scan(&reminder.ID, &reminder.CreatedAt)

	return err
}

func (r *ReminderRepository) GetPendingReminders() ([]models.Reminder, error) {
	query := `
	SELECT id, user_id, media_id, message, remind_at, sent, created_at
	FROM reminders
	WHERE sent = FALSE AND remind_at <= CURRENT_TIMESTAMP
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var newReminders models.Reminder
		err := rows.Scan(&newReminders.ID, &newReminders.UserID, &newReminders.MediaID, &newReminders.Message,
			&newReminders.RemindAt, &newReminders.Sent, &newReminders.CreatedAt)

		if err != nil {
			return nil, err
		}
		reminders = append(reminders, newReminders)
	}

	return reminders, nil
}

func (r *ReminderRepository) GetRemindersByUser(userID string) ([]models.Reminder, error) {
	query := `
	SELECT id, user_id, media_id, message, remind_at, sent, created_at
	FROM reminders
	WHERE user_id = $1
	ORDER BY remind_at ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.MediaID, &reminder.Message,
			&reminder.RemindAt, &reminder.Sent, &reminder.CreatedAt)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (r *ReminderRepository) MarkReminderAsSent(reminderID int) error {
	query := `UPDATE reminders SET sent = TRUE WHERE id = $1`
	_, err := r.db.Exec(query, reminderID)
	return err
}

// Repositories struct combines all repos
type Repositories struct {
	User      *UserRepository
	Media     *MediaRepository
	UserMedia *UserMediaRepository
	Reminder  *ReminderRepository
}

func NewRepositories(db *DB) *Repositories {
	return &Repositories{
		User:      NewUserRepository(db),
		Media:     NewMediaRepository(db),
		UserMedia: NewUserMediaRepository(db),
		Reminder:  NewReminderRepository(db),
	}
}

// TODO: Creeate another Telegram Bot
// Seeding not really enough for it...
// TODO: adapt some logic to handle duplicate entries
