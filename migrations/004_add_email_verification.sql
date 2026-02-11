-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN email_verification_token_hash TEXT,
    ADD COLUMN email_verification_expires_at TIMESTAMPTZ;

CREATE INDEX idx_users_email_verification_token_hash
    ON users (email_verification_token_hash)
    WHERE email_verification_token_hash IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_email_verification_token_hash;

ALTER TABLE users
    DROP COLUMN IF EXISTS email_verification_token_hash,
    DROP COLUMN IF EXISTS email_verification_expires_at;
-- +goose StatementEnd
