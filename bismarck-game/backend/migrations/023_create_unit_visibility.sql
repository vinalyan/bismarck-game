-- +goose Up
CREATE TABLE IF NOT EXISTS unit_visibility (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES naval_units(id) ON DELETE CASCADE,
    player_id VARCHAR(50) NOT NULL,
    visibility VARCHAR(20) NOT NULL,
    last_known_hex VARCHAR(10),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_unit_visibility_game_id ON unit_visibility(game_id);
CREATE INDEX IF NOT EXISTS idx_unit_visibility_player_id ON unit_visibility(player_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unit_visibility_game_unit_player
    ON unit_visibility(game_id, unit_id, player_id);

-- +goose Down
DROP TABLE IF EXISTS unit_visibility;

