-- Bismarck Game Database Initialization

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE
);

-- Games table
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    player1_id UUID REFERENCES users(id),
    player2_id UUID REFERENCES users(id),
    current_turn INTEGER DEFAULT 1,
    current_phase VARCHAR(20) DEFAULT 'waiting',
    status VARCHAR(20) DEFAULT 'waiting',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Game states table (for persistence)
CREATE TABLE IF NOT EXISTS game_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    state_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, turn, phase)
);

-- User sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Game turns table (for phase management)
CREATE TABLE IF NOT EXISTS game_turns (
    id VARCHAR(255) PRIMARY KEY,
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn_number INTEGER NOT NULL,
    current_phase VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, turn_number)
);

-- Phase records table (for tracking phase status)
CREATE TABLE IF NOT EXISTS phase_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn_number INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, turn_number, phase)
);

-- Movements table (for tracking unit movements)
CREATE TABLE IF NOT EXISTS movements (
    id VARCHAR(255) PRIMARY KEY,
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    unit_id VARCHAR(255) NOT NULL,
    from_hex VARCHAR(10) NOT NULL,
    to_hex VARCHAR(10) NOT NULL,
    path JSONB NOT NULL DEFAULT '[]',
    fuel_cost INTEGER NOT NULL DEFAULT 0,
    hexes_moved INTEGER NOT NULL DEFAULT 0,
    movement_type VARCHAR(20) NOT NULL DEFAULT 'normal',
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id);
CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id);
CREATE INDEX IF NOT EXISTS idx_game_states_game_id ON game_states(game_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_game_turns_game_id ON game_turns(game_id);
CREATE INDEX IF NOT EXISTS idx_game_turns_turn_number ON game_turns(turn_number);
CREATE INDEX IF NOT EXISTS idx_phase_records_game_id ON phase_records(game_id);
CREATE INDEX IF NOT EXISTS idx_phase_records_turn_phase ON phase_records(turn_number, phase);
CREATE INDEX IF NOT EXISTS idx_movements_game_id ON movements(game_id);
CREATE INDEX IF NOT EXISTS idx_movements_unit_id ON movements(unit_id);
CREATE INDEX IF NOT EXISTS idx_movements_turn ON movements(turn);