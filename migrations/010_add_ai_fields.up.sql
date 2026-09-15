-- +migrate Up
-- Add AI Player fields to profiles table: conduct rules, approach guidance,
-- AI settings, and one-time Safety and Abuse policy acceptance record.
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS conduct_rules TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS approach_guidance TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS ai_settings JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS policy_version_accepted TEXT,
ADD COLUMN IF NOT EXISTS policy_accepted_at TIMESTAMPTZ;
