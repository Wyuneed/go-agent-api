package valueobject

import "fmt"

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

func NewMessageRole(role string) (MessageRole, error) {
	switch MessageRole(role) {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return MessageRole(role), nil
	default:
		return "", fmt.Errorf("invalid message role: %s", role)
	}
}

func (r MessageRole) String() string {
	return string(r)
}

func (r MessageRole) IsAssistant() bool {
	return r == RoleAssistant
}

func (r MessageRole) IsTool() bool {
	return r == RoleTool
}
