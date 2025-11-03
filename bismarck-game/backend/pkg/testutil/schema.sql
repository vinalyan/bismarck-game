-- Тестовая схема БД для unit-тестов
-- Создание всех необходимых таблиц

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    last_login TIMESTAMP,
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
    turn_number INTEGER DEFAULT 0,
    current_phase VARCHAR(20) DEFAULT 'setup',
    status VARCHAR(20) DEFAULT 'waiting',
    victory_points JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица морских юнитов
CREATE TABLE IF NOT EXISTS naval_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    category VARCHAR(20) DEFAULT 'naval',
    class VARCHAR(50),
    owner VARCHAR(50) NOT NULL,
    nationality VARCHAR(20) DEFAULT 'german',
    position VARCHAR(10) NOT NULL,
    setup_hex VARCHAR(10),
    base_position VARCHAR(10),
    max_speed INTEGER,
    endurance INTEGER,
    evasion INTEGER DEFAULT 0,
    base_evasion INTEGER DEFAULT 0,
    primary_armament_bow INTEGER DEFAULT 0,
    primary_armament_stern INTEGER DEFAULT 0,
    secondary_armament INTEGER DEFAULT 0,
    base_primary_armament_bow INTEGER DEFAULT 0,
    base_primary_armament_stern INTEGER DEFAULT 0,
    base_secondary_armament INTEGER DEFAULT 0,
    torpedoes INTEGER DEFAULT 0,
    max_torpedoes INTEGER DEFAULT 0,
    radar_level INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    fuel INTEGER DEFAULT 0,
    max_fuel INTEGER DEFAULT 0,
    hull_boxes INTEGER DEFAULT 0,
    current_hull INTEGER DEFAULT 0,
    speed_rating VARCHAR(20),
    detection_level VARCHAR(20),
    last_known_pos VARCHAR(10),
    task_force_id UUID,
    damage JSONB DEFAULT '[]',
    previous_turn_moved_hexes INTEGER DEFAULT 0,
    last_move_turn INTEGER DEFAULT 0,
    no_movement_turns_left INTEGER DEFAULT 0,
    movement_used INTEGER DEFAULT 0,
    is_activated BOOLEAN DEFAULT false,
    is_emergency_fuel BOOLEAN DEFAULT false,
    emergency_turn INTEGER DEFAULT 0,
    is_patrolling BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица воздушных юнитов
CREATE TABLE IF NOT EXISTS air_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    name VARCHAR(100) NOT NULL,
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
    nationality VARCHAR(20) NOT NULL DEFAULT 'german',
    position VARCHAR(10) NOT NULL,
    speed INTEGER DEFAULT 0,
    units JSONB DEFAULT '[]',
    is_visible BOOLEAN DEFAULT true,
    detection_level VARCHAR(20) DEFAULT 'none',
    last_move_turn INTEGER DEFAULT 0,
    is_activated BOOLEAN DEFAULT false,
    is_patrolling BOOLEAN DEFAULT false,
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
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255),
    actor_name VARCHAR(255),
    target_id VARCHAR(255),
    target_name VARCHAR(255),
    description TEXT NOT NULL,
    visibility JSONB,
    data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица поиска юнитов
CREATE TABLE IF NOT EXISTS unit_searches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    unit_id UUID NOT NULL,
    target_hex VARCHAR(10) NOT NULL,
    search_type VARCHAR(20) NOT NULL,
    search_factors INTEGER DEFAULT 0,
    units_found JSONB DEFAULT '[]',
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
    path JSONB DEFAULT '[]',
    fuel_cost INTEGER DEFAULT 0,
    hexes_moved INTEGER NOT NULL,
    movement_type VARCHAR(20) NOT NULL,
    turn INTEGER DEFAULT 1,
    phase VARCHAR(20) DEFAULT 'movement',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

-- Таблица универсальных маркеров гексов
CREATE TABLE IF NOT EXISTS hex_markers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id UUID NOT NULL,
    hex_id VARCHAR(10) NOT NULL,
    marker_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hex_markers_game_hex_type ON hex_markers(game_id, hex_id, marker_type);
CREATE INDEX IF NOT EXISTS idx_hex_markers_game_player_type ON hex_markers(game_id, player_id, marker_type);
CREATE INDEX IF NOT EXISTS idx_hex_markers_game_type ON hex_markers(game_id, marker_type);
