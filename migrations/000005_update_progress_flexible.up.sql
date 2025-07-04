-- Update user_media table to support flexible progress tracking
ALTER TABLE user_media 
ADD COLUMN progress_current DECIMAL(10,2) DEFAULT 0,
ADD COLUMN progress_total DECIMAL(10,2) DEFAULT 0,
ADD COLUMN progress_unit VARCHAR(50) DEFAULT 'episodes',
ADD COLUMN progress_details TEXT DEFAULT '';

-- Migrate existing progress data
UPDATE user_media 
SET progress_current = progress::DECIMAL(10,2),
    progress_unit = 'episodes'
WHERE progress IS NOT NULL AND progress > 0;

-- Drop the old progress column
ALTER TABLE user_media DROP COLUMN progress; 