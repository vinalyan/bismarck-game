-- Migration: Add flight_path_search_markers table
-- Date: 2025-11-02
-- Description: Creates table for storing flight path search markers per hex

-- Create flight_path_search_markers table
CREATE TABLE IF NOT EXISTS flight_path_search_markers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id UUID NOT NULL,
    hex_id VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- One marker per hex per player
    UNIQUE(game_id, player_id, hex_id)
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_flight_path_search_markers_game_hex 
    ON flight_path_search_markers(game_id, hex_id);
    
CREATE INDEX IF NOT EXISTS idx_flight_path_search_markers_game_player 
    ON flight_path_search_markers(game_id, player_id);

-- Add comments
COMMENT ON TABLE flight_path_search_markers IS 'Stores flight path search markers placed on hexes. Each marker gives +2 search factors in its hex.';
COMMENT ON COLUMN flight_path_search_markers.game_id IS 'Game where the marker is placed';
COMMENT ON COLUMN flight_path_search_markers.player_id IS 'Player (side) that placed the marker';
COMMENT ON COLUMN flight_path_search_markers.hex_id IS 'Hex coordinate where the marker is placed';



