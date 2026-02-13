package response

// Swagger-only typed response wrappers.
// These structs are used exclusively in swag @Success/@Failure annotations
// so the generated OpenAPI spec has proper schemas instead of "any".

// ---- Shared error envelope ----

// SwaggerErrorResponse is returned on all error responses.
type SwaggerErrorResponse struct {
	Success bool        `json:"success" example:"false"`
	Error   *SwaggerErr `json:"error"`
}

// SwaggerErr contains the machine-readable error code and human message.
type SwaggerErr struct {
	Code    string `json:"code"    example:"INVALID_REQUEST"`
	Message string `json:"message" example:"Invalid request body"`
}

// ---- Health ----

// SwaggerHealthData is the data payload for /health and /ready.
type SwaggerHealthData struct {
	Status string `json:"status" example:"ok"`
}

// SwaggerHealthResponse wraps the health status.
type SwaggerHealthResponse struct {
	Success bool              `json:"success" example:"true"`
	Data    SwaggerHealthData `json:"data"`
}

// ---- Auth ----

// SwaggerUserData represents a user object returned in responses.
type SwaggerUserData struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email"      example:"user@example.com"`
	Name      string `json:"name"       example:"John Doe"`
	Role      string `json:"role"       example:"user"`
	IsActive  bool   `json:"is_active"  example:"true"`
	CreatedAt string `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// SwaggerRegisterData is the data payload returned after successful registration.
type SwaggerRegisterData struct {
	User SwaggerUserData `json:"user"`
}

// SwaggerRegisterResponse wraps the register result.
type SwaggerRegisterResponse struct {
	Success bool                `json:"success" example:"true"`
	Data    SwaggerRegisterData `json:"data"`
}

// SwaggerLoginData is the data payload returned after successful login.
type SwaggerLoginData struct {
	AccessToken  string          `json:"access_token"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string          `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string          `json:"token_type"    example:"Bearer"`
	ExpiresIn    int             `json:"expires_in"    example:"900"`
	User         SwaggerUserData `json:"user"`
}

// SwaggerLoginResponse wraps the login result.
type SwaggerLoginResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    SwaggerLoginData `json:"data"`
}

// ---- Chat / Conversations ----

// SwaggerConversationData represents a conversation object.
type SwaggerConversationData struct {
	ID           string `json:"id"             example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string `json:"user_id"        example:"550e8400-e29b-41d4-a716-446655440000"`
	Title        string `json:"title"          example:"My first conversation"`
	Status       string `json:"status"         example:"active"`
	AgentType    string `json:"agent_type"     example:"general"`
	Model        string `json:"model"          example:"gpt-4o-mini"`
	MessageCount int    `json:"message_count"  example:"5"`
	TotalTokens  int    `json:"total_tokens"   example:"1024"`
	CreatedAt    string `json:"created_at"     example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string `json:"updated_at"     example:"2024-01-01T00:01:00Z"`
}

// SwaggerMessageData represents a single message in a conversation.
type SwaggerMessageData struct {
	ID             string `json:"id"              example:"550e8400-e29b-41d4-a716-446655440000"`
	ConversationID string `json:"conversation_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role           string `json:"role"            example:"assistant"`
	Content        string `json:"content"         example:"Hello! How can I help you today?"`
	Model          string `json:"model"           example:"gpt-4o-mini"`
	CreatedAt      string `json:"created_at"      example:"2024-01-01T00:00:00Z"`
}

// SwaggerSendMessageData is returned after sending a message.
type SwaggerSendMessageData struct {
	Conversation SwaggerConversationData `json:"conversation"`
	Message      SwaggerMessageData      `json:"message"`
}

// SwaggerSendMessageResponse wraps the send-message result.
type SwaggerSendMessageResponse struct {
	Success bool                   `json:"success" example:"true"`
	Data    SwaggerSendMessageData `json:"data"`
}

// SwaggerConversationResponse wraps a single conversation result.
type SwaggerConversationResponse struct {
	Success bool                    `json:"success" example:"true"`
	Data    SwaggerConversationData `json:"data"`
}

// SwaggerConversationListResponse wraps a list of conversations.
type SwaggerConversationListResponse struct {
	Success bool                      `json:"success" example:"true"`
	Data    []SwaggerConversationData `json:"data"`
}

// SwaggerApprovalData is returned after an approval action.
type SwaggerApprovalData struct {
	Status  string `json:"status"  example:"approved"`
	Message string `json:"message" example:"Action approved, resuming workflow"`
}

// SwaggerApprovalResponse wraps the approval result.
type SwaggerApprovalResponse struct {
	Success bool                `json:"success" example:"true"`
	Data    SwaggerApprovalData `json:"data"`
}

// SwaggerChatCompletionChoice represents one completion choice.
type SwaggerChatCompletionChoice struct {
	Index        int                `json:"index"         example:"0"`
	Message      SwaggerMessageData `json:"message"`
	FinishReason string             `json:"finish_reason" example:"stop"`
}

// SwaggerChatCompletionData is the OpenAI-compatible response payload.
type SwaggerChatCompletionData struct {
	ID      string                        `json:"id"      example:"chatcmpl-abc123"`
	Object  string                        `json:"object"  example:"chat.completion"`
	Model   string                        `json:"model"   example:"gpt-4o-mini"`
	Choices []SwaggerChatCompletionChoice `json:"choices"`
}

// SwaggerChatCompletionResponse wraps a chat completion result.
type SwaggerChatCompletionResponse struct {
	Success bool                      `json:"success" example:"true"`
	Data    SwaggerChatCompletionData `json:"data"`
}

// ---- Tools ----

// SwaggerToolFunction represents a tool's function definition.
type SwaggerToolFunction struct {
	Name        string `json:"name"        example:"calculator"`
	Description string `json:"description" example:"Perform mathematical calculations"`
}

// SwaggerToolData represents a single available tool.
type SwaggerToolData struct {
	Type     string              `json:"type"     example:"function"`
	Function SwaggerToolFunction `json:"function"`
}

// SwaggerToolsData is the data payload for listing tools.
type SwaggerToolsData struct {
	Tools []SwaggerToolData `json:"tools"`
}

// SwaggerToolsResponse wraps the tools list.
type SwaggerToolsResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    SwaggerToolsData `json:"data"`
}

// SwaggerToolResultResponse wraps a single tool execution result.
type SwaggerToolResultResponse struct {
	Success bool `json:"success" example:"true"`
	Data    any  `json:"data"`
}

// SwaggerBatchResultData holds parallel tool execution results.
type SwaggerBatchResultData struct {
	Results []any `json:"results"`
}

// SwaggerBatchResultResponse wraps batch tool execution results.
type SwaggerBatchResultResponse struct {
	Success bool                   `json:"success" example:"true"`
	Data    SwaggerBatchResultData `json:"data"`
}
