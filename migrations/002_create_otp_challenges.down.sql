-- +migrate Down
-- Drop otp_challenges table
DROP TABLE IF EXISTS otp_challenges CASCADE;
