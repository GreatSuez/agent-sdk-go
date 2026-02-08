package api

import (
	"context"
	"time"
)

type AuditLog struct {
	ActorKeyID string
	Action     string
	Resource   string
	Payload    string
}

type AuditLogEntry struct {
	ID         int64     `json:"id"`
	ActorKeyID string    `json:"actorKeyId"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AuditStore interface {
	Record(ctx context.Context, entry AuditLog) error
	Close() error
}

type AuditReader interface {
	AuditStore
	List(ctx context.Context, limit int, offset int) ([]AuditLogEntry, error)
}
