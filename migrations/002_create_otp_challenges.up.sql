-- +migrate Up
-- Create otp_challenges table for SP01 authentication
CREATE TABLE IF NOT EXISTS otp_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    otp_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on email for faster lookups and rate limiting
CREATE INDEX IF NOT EXISTS idx_otp_challenges_email ON otp_challenges(email);

-- Create index on expires_at for cleanup of expired OTPs
CREATE INDEX IF NOT EXISTS idx_otp_challenges_expires_at ON otp_challenges(expires_at);

-- +migrate Down
DROP TABLE IF EXISTS otp_challenges CASCADE;
