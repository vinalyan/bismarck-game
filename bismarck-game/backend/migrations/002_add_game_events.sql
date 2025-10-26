-- Game events table (for game log)
CREATE TABLE IF NOT EXISTS game_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'movement', 'phase_change', 'turn_change'
    actor_id VARCHAR(255), -- ID юнита или игрока, который совершил действие
    actor_name VARCHAR(255), -- Имя юнита или игрока
    description TEXT NOT NULL, -- Описание события
    data JSONB, -- Дополнительные данные события
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_game_events_game_id ON game_events(game_id);
CREATE INDEX IF NOT EXISTS idx_game_events_turn ON game_events(turn);
CREATE INDEX IF NOT EXISTS idx_game_events_created_at ON game_events(created_at);
CREATE INDEX IF NOT EXISTS idx_game_events_event_type ON game_events(event_type);
