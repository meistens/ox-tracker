CREATE TABLE IF NOT EXISTS user_media (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users (id) ON DELETE CASCADE,
    media_id INTEGER REFERENCES media (id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    progress INTEGER DEFAULT 0,
    rating DECIMAL(3, 1),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, media_id)
);
