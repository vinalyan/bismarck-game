-- Bismarck Game Database Initialization

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'player',
    stats JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true
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
    completed_at TIMESTAMP WITH TIME ZONE,
    winner UUID REFERENCES users(id),
    victory_type VARCHAR(20),
    started_at TIMESTAMP WITH TIME ZONE,
    last_action_at TIMESTAMP WITH TIME ZONE
);

-- Game states table (for persistence)
CREATE TABLE IF NOT EXISTS game_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    state_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    sequence INTEGER DEFAULT 0,
    checksum VARCHAR(255)
);

-- User sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true
);

-- User preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'dark',
    language VARCHAR(10) DEFAULT 'en',
    notifications BOOLEAN DEFAULT true,
    sound_enabled BOOLEAN DEFAULT true,
    auto_save BOOLEAN DEFAULT true,
    show_tutorials BOOLEAN DEFAULT true,
    default_game_mode VARCHAR(20) DEFAULT 'standard',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User achievements table
CREATE TABLE IF NOT EXISTS user_achievements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    achievement VARCHAR(100) NOT NULL,
    unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    progress INTEGER DEFAULT 0,
    max_progress INTEGER DEFAULT 0,
    UNIQUE(user_id, achievement)
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
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id);
CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id);
CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);

CREATE INDEX IF NOT EXISTS idx_game_states_game_id ON game_states(game_id);
CREATE INDEX IF NOT EXISTS idx_game_states_turn_phase ON game_states(turn, phase);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);

CREATE INDEX IF NOT EXISTS idx_user_achievements_user_id ON user_achievements(user_id);
CREATE INDEX IF NOT EXISTS idx_user_achievements_achievement ON user_achievements(achievement);