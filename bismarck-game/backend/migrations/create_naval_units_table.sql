-- Создание таблицы naval_units
CREATE TABLE IF NOT EXISTS naval_units (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(10) NOT NULL,
    class VARCHAR(50),
    owner VARCHAR(100) NOT NULL,
    nationality VARCHAR(20),
    position VARCHAR(10),
    setup_hex VARCHAR(10),
    evasion INTEGER DEFAULT 0,
    base_evasion INTEGER DEFAULT 0,
    speed_rating VARCHAR(10),
    fuel INTEGER DEFAULT 0,
    max_fuel INTEGER DEFAULT 0,
    hull_boxes INTEGER DEFAULT 0,
    current_hull INTEGER DEFAULT 0,
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
    detection_level VARCHAR(20) DEFAULT 'none',
    damage JSONB DEFAULT '[]'::jsonb,
    last_known_pos VARCHAR(10),
    task_force_id UUID,
    previous_turn_moved_hexes INTEGER DEFAULT 0,
    last_move_turn INTEGER DEFAULT 0,
    movement_used INTEGER DEFAULT 0,
    no_movement_turns_left INTEGER DEFAULT 0,
    is_emergency_fuel BOOLEAN DEFAULT false,
    emergency_removal_turn INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_naval_units_game_id ON naval_units(game_id);
CREATE INDEX IF NOT EXISTS idx_naval_units_owner ON naval_units(owner);
CREATE INDEX IF NOT EXISTS idx_naval_units_status ON naval_units(status);
CREATE INDEX IF NOT EXISTS idx_naval_units_task_force_id ON naval_units(task_force_id);
CREATE INDEX IF NOT EXISTS idx_naval_units_position ON naval_units(position);
CREATE INDEX IF NOT EXISTS idx_naval_units_emergency_removal_turn ON naval_units(emergency_removal_turn);
CREATE INDEX IF NOT EXISTS idx_naval_units_is_emergency_fuel ON naval_units(is_emergency_fuel);
