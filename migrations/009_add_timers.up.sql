-- +migrate Up
-- Add timers column to profiles table for time-based automation
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS timers JSONB NOT NULL DEFAULT '{"items": []}'::jsonb;

-- +migrate Down
-- Remove timers column from profiles table
ALTER TABLE profiles
DROP COLUMN IF EXISTS timers;
