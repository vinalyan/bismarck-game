-- Migration: Add missing columns to existing tables
-- This migration adds the missing columns that were identified during API testing

-- Add missing columns to games table
ALTER TABLE games ADD COLUMN IF NOT EXISTS started_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE games ADD COLUMN IF NOT EXISTS last_action_at TIMESTAMP WITH TIME ZONE;

-- Add missing column to user_sessions table  
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Add indexes for better performance on new columns
CREATE INDEX IF NOT EXISTS idx_games_started_at ON games(started_at);
CREATE INDEX IF NOT EXISTS idx_games_last_action_at ON games(last_action_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);
