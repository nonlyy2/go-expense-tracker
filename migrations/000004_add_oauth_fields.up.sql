ALTER TABLE users
    ADD COLUMN IF NOT EXISTS oauth_provider VARCHAR(20) DEFAULT '',
    ADD COLUMN IF NOT EXISTS oauth_id VARCHAR(255) DEFAULT '';

ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider != '';