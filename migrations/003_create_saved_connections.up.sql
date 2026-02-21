-- +migrate Up
-- Create saved_connections table for SP02 session proxy
-- Stores user's saved MUD server connections
CREATE TABLE IF NOT EXISTS saved_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255),
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 23,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on user_id for faster lookups when listing user's connections
CREATE INDEX IF NOT EXISTS idx_saved_connections_user_id ON saved_connections(user_id);

-- Create index on user_id and name for faster lookups when checking duplicate names
CREATE INDEX IF NOT EXISTS idx_saved_connections_user_id_name ON saved_connections(user_id, name);
