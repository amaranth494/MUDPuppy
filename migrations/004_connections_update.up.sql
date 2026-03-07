-- +migrate Up
-- Add protocol and last_connected_at columns to saved_connections table
ALTER TABLE saved_connections ADD COLUMN IF NOT EXISTS protocol VARCHAR(50) NOT NULL DEFAULT 'telnet';
ALTER TABLE saved_connections ADD COLUMN IF NOT EXISTS last_connected_at TIMESTAMP WITH TIME ZONE;

-- Create index for recent connections query
CREATE INDEX IF NOT EXISTS idx_saved_connections_last_connected_at ON saved_connections(last_connected_at DESC NULLS LAST);
