-- Add missing fields to game_events table
ALTER TABLE game_events 
ADD COLUMN IF NOT EXISTS target_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS target_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS visibility JSONB;
