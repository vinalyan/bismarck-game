-- Migration: Add missing columns to naval_units table
-- This migration adds all the missing columns needed for the naval units functionality

-- Add missing columns to naval_units table
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS nationality VARCHAR(20);
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS setup_hex VARCHAR(10);
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS evasion INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS base_evasion INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS speed_rating VARCHAR(10);
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS hull_boxes INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS current_hull INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS primary_armament_bow INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS primary_armament_stern INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS secondary_armament INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS base_primary_armament_bow INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS base_primary_armament_stern INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS base_secondary_armament INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS torpedoes INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS max_torpedoes INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS radar_level INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS detection_level VARCHAR(20) DEFAULT 'none';
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS damage JSONB DEFAULT '[]';
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS last_known_pos VARCHAR(10);
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS task_force_id UUID;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS previous_turn_moved_hexes INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS last_move_turn INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS movement_used INTEGER DEFAULT 0;
ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS no_movement_turns_left INTEGER DEFAULT 0;

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_naval_units_game_id ON naval_units(game_id);
CREATE INDEX IF NOT EXISTS idx_naval_units_owner ON naval_units(owner);
CREATE INDEX IF NOT EXISTS idx_naval_units_position ON naval_units(position);
CREATE INDEX IF NOT EXISTS idx_naval_units_status ON naval_units(status);
CREATE INDEX IF NOT EXISTS idx_naval_units_task_force_id ON naval_units(task_force_id);

