-- +migrate Up
-- Create profiles table for per-connection settings and keybindings
-- Requires pgcrypto extension for gen_random_uuid()

-- Ensure pgcrypto extension is available
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create profiles table
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    keybindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    settings JSONB NOT NULL DEFAULT '{"scrollback_limit": 1000, "echo_input": false, "timestamp_output": false, "word_wrap": true}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create unique index for connection_id (1:1 relationship)
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_connection_id ON profiles(connection_id);

-- Create index for user_id lookups
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);
