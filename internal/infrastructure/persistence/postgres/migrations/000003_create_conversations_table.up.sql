CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'pending_approval', 'completed', 'failed', 'archived')),

    -- Agent configuration
    agent_type VARCHAR(100) DEFAULT 'general',
    system_prompt TEXT,
    model VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,

    -- Workflow state (Eino)
    current_node VARCHAR(100),
    workflow_state JSONB DEFAULT '{}',

    -- Counters
    message_count INT DEFAULT 0,
    tool_call_count INT DEFAULT 0,
    total_tokens INT DEFAULT 0,
    total_cost DECIMAL(10,6) DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE  -- Soft delete
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_user_status ON conversations(user_id, status);
CREATE INDEX idx_conversations_updated ON conversations(updated_at DESC);
CREATE INDEX idx_conversations_last_message ON conversations(last_message_at DESC NULLS LAST);
CREATE INDEX idx_conversations_pending ON conversations(status) WHERE status = 'pending_approval';
CREATE INDEX idx_conversations_active ON conversations(user_id, updated_at DESC)
    WHERE deleted_at IS NULL AND status IN ('active', 'pending_approval');
CREATE INDEX idx_conversations_tags ON conversations USING GIN(tags);

CREATE TRIGGER update_conversations_updated_at BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
