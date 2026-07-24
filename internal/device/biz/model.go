package biz

import (
	"errors"
	"time"
)

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceOffline       = errors.New("device offline")
	ErrDeviceMaintenance   = errors.New("device in maintenance")
	ErrDeviceDisabled      = errors.New("device disabled")
	ErrCommandNotFound     = errors.New("command not found")
	ErrCommandExpired      = errors.New("command expired")
	ErrMaxAttempts         = errors.New("maximum command attempts exceeded")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
	ErrGatewayOffline      = ErrDeviceOffline
	ErrGatewayTimeout      = errors.New("gateway timeout")
	ErrGatewayFailure      = errors.New("gateway failure")
	ErrCommandBusy         = errors.New("command is already running")
	ErrInvalidAction       = errors.New("invalid device action")
	ErrIdempotencyRequired = errors.New("idempotency key required")
)

const (
	DefaultCommandTTL  = time.Minute
	MaxCommandAttempts = 3
)

type NetworkStatus string

const (
	NetworkOnline  NetworkStatus = "ONLINE"
	NetworkOffline NetworkStatus = "OFFLINE"
)

type OperationalStatus string

const (
	OperationalActive      OperationalStatus = "ACTIVE"
	OperationalMaintenance OperationalStatus = "MAINTENANCE"
	OperationalDisabled    OperationalStatus = "DISABLED"
)

type Device struct {
	ID                uint64
	DeviceNo          string
	NetworkStatus     NetworkStatus
	OperationalStatus OperationalStatus
	LastHeartbeatAt   time.Time
}

type Action string

const (
	ActionOpenDoor    Action = "OPEN_DOOR"
	ActionQueryStatus Action = "QUERY_STATUS"
)

type CommandStatus string

const (
	CommandPending   CommandStatus = "PENDING"
	CommandRunning   CommandStatus = "RUNNING"
	CommandSucceeded CommandStatus = "SUCCEEDED"
	CommandFailed    CommandStatus = "FAILED"
	CommandTimedOut  CommandStatus = "TIMED_OUT"
	CommandExpired   CommandStatus = "EXPIRED"
)

type CommandRequest struct {
	DeviceNo       string
	Action         Action
	CellNo         string
	Payload        []byte
	IdempotencyKey string
	TTL            time.Duration
	ExpiresAt      time.Time
}

type Command struct {
	ID             uint64
	CommandNo      string
	DeviceNo       string
	Action         Action
	CellNo         string
	Payload        []byte
	IdempotencyKey string
	Status         CommandStatus
	ExpiresAt      time.Time
	AttemptCount   int
	Result         GatewayResult
	ErrorCode      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
