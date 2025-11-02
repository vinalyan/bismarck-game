-- Migration: Add search markers fields
-- Date: 2025-11-02
-- Description: Adds fields for storing search markers (patrol markers and flight path search markers)

-- Add is_patrolling field to naval_units (for patrol markers +3 search factors)
ALTER TABLE naval_units 
ADD COLUMN IF NOT EXISTS is_patrolling BOOLEAN DEFAULT FALSE;

-- Add flight_path_search_hexes field to air_units (JSONB array of hex IDs with flight path search markers +2 search factors each)
ALTER TABLE air_units 
ADD COLUMN IF NOT EXISTS flight_path_search_hexes JSONB DEFAULT '[]'::jsonb;

-- Create index on is_patrolling for faster queries
CREATE INDEX IF NOT EXISTS idx_naval_units_is_patrolling ON naval_units(is_patrolling) WHERE is_patrolling = true;

-- Create GIN index on flight_path_search_hexes for faster JSONB queries
CREATE INDEX IF NOT EXISTS idx_air_units_flight_path_search_hexes ON air_units USING GIN (flight_path_search_hexes);

-- Add comments to columns
COMMENT ON COLUMN naval_units.is_patrolling IS 'Indicates if the unit is patrolling (gives +3 search factors in its hex)';
COMMENT ON COLUMN air_units.flight_path_search_hexes IS 'Array of hex IDs where flight path search markers are placed (each gives +2 search factors)';

