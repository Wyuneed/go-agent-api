package event

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	OccurredAt() time.Time
}

// baseEvent contains common fields for all domain events.
type baseEvent struct {
	ID        uuid.UUID
	Type      string
	Timestamp time.Time
}

func newBaseEvent(eventType string) baseEvent {
	return baseEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Timestamp: time.Now(),
	}
}

func (e baseEvent) EventID() uuid.UUID    { return e.ID }
func (e baseEvent) EventType() string      { return e.Type }
func (e baseEvent) OccurredAt() time.Time  { return e.Timestamp }
