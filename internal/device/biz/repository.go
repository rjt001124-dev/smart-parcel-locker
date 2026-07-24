package biz

import (
	"context"
	"time"
)

// Repository is the persistence boundary implemented by the MySQL adapter.
// MarkRunning must atomically check PENDING, expiry, and attempt limits.
type Repository interface {
	UpdateHeartbeat(context.Context, string, time.Time, bool) error
	RecordHeartbeat(context.Context, DeviceHeartbeat) error
	FindDevice(context.Context, string) (Device, error)
	FindCommandByIdempotencyKey(context.Context, string) (Command, error)
	FindCommand(context.Context, string) (Command, error)
	CreateCommand(context.Context, Command) (Command, error)
	MarkRunning(context.Context, string, time.Time) (bool, error)
	CompleteCommand(context.Context, string, CommandStatus, GatewayResult, string) error
}
