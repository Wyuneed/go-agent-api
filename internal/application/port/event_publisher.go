package port

import (
	"context"

	"github.com/wyuneed/go-agent-api/internal/domain/event"
)

// EventPublisher is the interface for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, event event.DomainEvent) error
}
