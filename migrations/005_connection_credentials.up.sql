-- +migrate Up
-- Create connection_credentials table for storing encrypted credentials
CREATE TABLE IF NOT EXISTS connection_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    username VARCHAR(255),
    encrypted_password BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    auto_login BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create unique index for connection_id (one credential set per connection)
CREATE UNIQUE INDEX IF NOT EXISTS idx_connection_credentials_connection_id ON connection_credentials(connection_id);
