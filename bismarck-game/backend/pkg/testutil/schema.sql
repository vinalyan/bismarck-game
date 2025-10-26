-- Тестовая схема БД для unit-тестов
-- Создание всех необходимых таблиц

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица игр
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    player1_id UUID REFERENCES users(id),
    player2_id UUID REFERENCES users(id),
    current_turn INTEGER DEFAULT 0,
    current_phase VARCHAR(20) DEFAULT 'setup',
    status VARCHAR(20) DEFAULT 'waiting',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица морских юнитов
CREATE TABLE IF NOT EXISTS naval_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    owner VARCHAR(50) NOT NULL,
    position VARCHAR(10) NOT NULL,
    base_position VARCHAR(10),
    max_speed INTEGER,
    endurance INTEGER,
    status VARCHAR(20) DEFAULT 'active',
    fuel INTEGER DEFAULT 0,
    max_fuel INTEGER DEFAULT 0,
    hull_boxes INTEGER DEFAULT 0,
    current_hull INTEGER DEFAULT 0,
    speed_rating VARCHAR(20),
    detection_level VARCHAR(20),
    damage JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица воздушных юнитов
CREATE TABLE IF NOT EXISTS air_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    type VARCHAR(20) NOT NULL,
    owner VARCHAR(50) NOT NULL,
    position VARCHAR(10) NOT NULL,
    base_position VARCHAR(10),
    max_speed INTEGER,
    endurance INTEGER,
    status VARCHAR(20) DEFAULT 'operational',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица оперативных соединений
CREATE TABLE IF NOT EXISTS task_forces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    name VARCHAR(100) NOT NULL,
    owner VARCHAR(50) NOT NULL,
    position VARCHAR(10) NOT NULL,
    is_visible BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица юнитов в соединениях
CREATE TABLE IF NOT EXISTS task_force_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_force_id UUID NOT NULL REFERENCES task_forces(id),
    unit_id UUID NOT NULL,
    unit_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица видимости юнитов
CREATE TABLE IF NOT EXISTS unit_visibility (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    unit_id UUID NOT NULL,
    player_id VARCHAR(50) NOT NULL,
    visibility VARCHAR(20) NOT NULL,
    last_known_hex VARCHAR(10),
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица событий игры
CREATE TABLE IF NOT EXISTS game_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица поиска юнитов
CREATE TABLE IF NOT EXISTS unit_searches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    unit_id UUID NOT NULL,
    target_hex VARCHAR(10) NOT NULL,
    result VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица движений
CREATE TABLE IF NOT EXISTS movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    unit_id UUID NOT NULL,
    from_hex VARCHAR(10) NOT NULL,
    to_hex VARCHAR(10) NOT NULL,
    hexes_moved INTEGER NOT NULL,
    movement_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_naval_units_game_id ON naval_units(game_id);
CREATE INDEX IF NOT EXISTS idx_naval_units_owner ON naval_units(owner);
CREATE INDEX IF NOT EXISTS idx_naval_units_position ON naval_units(position);
CREATE INDEX IF NOT EXISTS idx_air_units_game_id ON air_units(game_id);
CREATE INDEX IF NOT EXISTS idx_air_units_owner ON air_units(owner);
CREATE INDEX IF NOT EXISTS idx_air_units_position ON air_units(position);
CREATE INDEX IF NOT EXISTS idx_task_forces_game_id ON task_forces(game_id);
CREATE INDEX IF NOT EXISTS idx_task_forces_owner ON task_forces(owner);
CREATE INDEX IF NOT EXISTS idx_unit_visibility_game_id ON unit_visibility(game_id);
CREATE INDEX IF NOT EXISTS idx_unit_visibility_player_id ON unit_visibility(player_id);
CREATE INDEX IF NOT EXISTS idx_game_events_game_id ON game_events(game_id);
CREATE INDEX IF NOT EXISTS idx_unit_searches_game_id ON unit_searches(game_id);
CREATE INDEX IF NOT EXISTS idx_movements_game_id ON movements(game_id);
