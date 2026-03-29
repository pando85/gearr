-- Add api_tokens table for API token authentication

CREATE TABLE IF NOT EXISTS api_tokens (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    scope VARCHAR(20) NOT NULL DEFAULT 'read',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    last_used TIMESTAMP,
    created_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_api_tokens_scope ON api_tokens(scope);