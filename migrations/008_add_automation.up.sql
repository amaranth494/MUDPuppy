-- +migrate Up
-- Add automation fields to profiles table for aliases, triggers, and environment variables
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS aliases JSONB NOT NULL DEFAULT '{"items": []}'::jsonb,
ADD COLUMN IF NOT EXISTS triggers JSONB NOT NULL DEFAULT '{"items": []}'::jsonb,
ADD COLUMN IF NOT EXISTS variables JSONB NOT NULL DEFAULT '{"items": []}'::jsonb;
