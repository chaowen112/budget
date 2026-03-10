package model

import (
	"time"

	"github.com/google/uuid"
)

// ApiKey represents a user's API key
type ApiKey struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	KeyValue   string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}
