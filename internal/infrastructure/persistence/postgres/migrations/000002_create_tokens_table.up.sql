CREATE TABLE user_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    token_type VARCHAR(50) NOT NULL DEFAULT 'api_key' CHECK (token_type IN ('api_key', 'access', 'refresh')),
    name VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    rate_limit_per_minute INT DEFAULT 60,
    rate_limit_per_day INT DEFAULT 10000,
    allowed_tools TEXT[],
    allowed_models TEXT[],
    is_revoked BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tokens_user_id ON user_tokens(user_id);
CREATE INDEX idx_tokens_token_hash ON user_tokens(token_hash);
CREATE INDEX idx_tokens_expires_at ON user_tokens(expires_at);
CREATE INDEX idx_tokens_valid ON user_tokens(token_hash)
    WHERE is_revoked = false;

CREATE TRIGGER update_tokens_updated_at BEFORE UPDATE ON user_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
