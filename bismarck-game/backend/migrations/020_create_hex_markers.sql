-- Универсальная таблица для всех типов маркеров на гексах
CREATE TABLE IF NOT EXISTS hex_markers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id UUID NOT NULL,
    hex_id VARCHAR(10) NOT NULL,
    marker_type VARCHAR(20) NOT NULL, -- 'flight_path_search', 'air_attack', etc.
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_hex_markers_game_hex_type 
    ON hex_markers(game_id, hex_id, marker_type);
    
CREATE INDEX IF NOT EXISTS idx_hex_markers_game_player_type 
    ON hex_markers(game_id, player_id, marker_type);

CREATE INDEX IF NOT EXISTS idx_hex_markers_game_type 
    ON hex_markers(game_id, marker_type);

-- Комментарии
COMMENT ON TABLE hex_markers IS 'Универсальная таблица для хранения маркеров на гексах. Поддерживает несколько маркеров одного типа в гексе.';
COMMENT ON COLUMN hex_markers.marker_type IS 'Тип маркера: flight_path_search (путь полета поиска), air_attack (воздушная атака) и т.д.';

