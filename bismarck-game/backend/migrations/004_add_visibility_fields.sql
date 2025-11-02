-- Migration: Add visibility fields to games table
-- Date: 2025-01-XX
-- Description: Adds visibility_level, is_fog, and weather_track fields to games table for visibility system

-- Add visibility_level field (1-10, where 10 = X = maximum visibility blocking all actions)
ALTER TABLE games 
ADD COLUMN IF NOT EXISTS visibility_level INTEGER DEFAULT 1 CHECK (visibility_level >= 1 AND visibility_level <= 10);

-- Add is_fog field (indicates fog weather condition)
ALTER TABLE games 
ADD COLUMN IF NOT EXISTS is_fog BOOLEAN DEFAULT FALSE;

-- Add weather_track field (0-9, weather track position)
ALTER TABLE games 
ADD COLUMN IF NOT EXISTS weather_track INTEGER DEFAULT 0 CHECK (weather_track >= 0 AND weather_track <= 9);

-- Create index on visibility_level for faster queries
CREATE INDEX IF NOT EXISTS idx_games_visibility_level ON games(visibility_level);

-- Create index on is_fog for faster queries
CREATE INDEX IF NOT EXISTS idx_games_is_fog ON games(is_fog);

-- Add comment to columns
COMMENT ON COLUMN games.visibility_level IS 'Current visibility level (1-10, where 10 = X blocks all actions). Formula: weather_track + time_of_day_modifier';
COMMENT ON COLUMN games.is_fog IS 'Indicates fog weather condition (weather_track 5-9). Blocks search, pursuit, and combat in fog hexes';
COMMENT ON COLUMN games.weather_track IS 'Weather track position (0-9). Values 5-9 indicate fog conditions';

