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

func (r *MediaRepository) CreateMedia(media *models.Media) error {
	query := `
	INSERT INTO media (external_id, title, type, description, release_date, poster_url, rating)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (external_id) DO NOTHING
	RETURNING id, created_at
	`
	err := r.db.QueryRow(query, media.ExternalID, media.Title, media.Type, media.Description, media.ReleaseDate, media.PosterURL, media.Rating).Scan(&media.ID, &media.CreatedAt)

	if err == sql.ErrNoRows {
		// TODO: logic for log/whatever to return
		return nil
	}

	return err
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
