CREATE TABLE IF NOT EXISTS media (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    release_date VARCHAR(50),
    poster_url VARCHAR(500),
    rating DECIMAL(3, 1),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
