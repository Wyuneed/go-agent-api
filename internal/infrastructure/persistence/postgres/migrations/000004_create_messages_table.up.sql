CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT,
    name VARCHAR(100),

    -- Tool-related
    tool_calls JSONB,
    tool_call_id VARCHAR(100),

    -- Metadata
    model VARCHAR(100),
    prompt_tokens INT DEFAULT 0,
    completion_tokens INT DEFAULT 0,
    latency_ms INT,

    -- Streaming
    is_streaming BOOLEAN DEFAULT false,
    stream_complete BOOLEAN DEFAULT true,

    -- Ordering
    sequence_number SERIAL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, sequence_number);
CREATE INDEX idx_messages_conversation_recent ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_role ON messages(conversation_id, role);
CREATE INDEX idx_messages_tool_calls ON messages(conversation_id) WHERE tool_calls IS NOT NULL;
