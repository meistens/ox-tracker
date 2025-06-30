package db

import "mtracker/internal/models"

// UserRepository handles user-related database ops
type UserRepository struct {
	db *DB
}

// MediaRepository handles media-related database ops
type MediaRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create User
func (r *UserRepository) Create(user *models.User) error {
	query := `
	INSERT INTO users (id, username, platform, uodated_at)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	ON CONFLICT (id) DO UPDATE SET
	username = $2, platform = $3, updated_at = CURRENT_TIMESTAMP
	RETURNING created_at`

	err := r.db.QueryRow(query, user.ID, user.Username, user.Platform).Scan(&user.CreatedAt)
	return err
}

// Get User by ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
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
