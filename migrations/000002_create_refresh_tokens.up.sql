-- Create refresh_tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

-- Index for user lookup
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index for token validation
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
