CREATE TABLE tool_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id UUID REFERENCES user_tokens(id) ON DELETE SET NULL,

    tool_name VARCHAR(255) NOT NULL,
    tool_call_id VARCHAR(100),
    input JSONB NOT NULL,
    output JSONB,

    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled', 'requires_approval')),
    error_message TEXT,

    -- Timing
    duration_ms INT,
    queued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Approval
    requires_approval BOOLEAN DEFAULT false,
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tool_exec_conversation ON tool_executions(conversation_id);
CREATE INDEX idx_tool_exec_user ON tool_executions(user_id);
CREATE INDEX idx_tool_exec_status ON tool_executions(status);
CREATE INDEX idx_tool_exec_tool_name ON tool_executions(tool_name);
CREATE INDEX idx_tool_exec_pending_approval ON tool_executions(status)
    WHERE status = 'requires_approval';
CREATE INDEX idx_tool_exec_created ON tool_executions(created_at DESC);
