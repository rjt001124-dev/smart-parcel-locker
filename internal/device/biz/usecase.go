package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

type UseCase struct {
	repo             Repository
	gateway          Gateway
	now              func() time.Time
	offlineThreshold time.Duration
	commandSequence  uint64
}

const terminalPersistenceTimeout = 5 * time.Second

// NewUseCase accepts an optional clock and offline threshold for deterministic tests.
func NewUseCase(repo Repository, gateway Gateway, options ...interface{}) *UseCase {
	uc := &UseCase{repo: repo, gateway: gateway, now: func() time.Time { return time.Now().UTC() }, offlineThreshold: 30 * time.Second}
	for _, option := range options {
		switch value := option.(type) {
		case func() time.Time:
			uc.now = value
		case time.Duration:
			if value > 0 {
				uc.offlineThreshold = value
			}
		}
	}
	return uc
}

func (uc *UseCase) Execute(ctx context.Context, req CommandRequest) (Command, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return Command{}, ErrIdempotencyRequired
	}
	if !validAction(req.Action) {
		return Command{}, ErrInvalidAction
	}
	if existing, err := uc.repo.FindCommandByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
		if !sameCommandRequest(existing, req) {
			return existing, ErrIdempotencyConflict
		}
		if existing.Status != CommandPending {
			return existing, nil
		}
		if existing.AttemptCount >= MaxCommandAttempts {
			return existing, ErrMaxAttempts
		}
		if !uc.now().UTC().Before(existing.ExpiresAt) {
			return uc.expire(ctx, existing)
		}
		return uc.run(ctx, existing)
	} else if !errors.Is(err, ErrCommandNotFound) {
		return Command{}, err
	}

	device, err := uc.repo.FindDevice(ctx, req.DeviceNo)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			return Command{}, ErrDeviceNotFound
		}
		return Command{}, err
	}
	if err := uc.validateDevice(device); err != nil {
		return Command{}, err
	}
	now := uc.now().UTC()
	expiresAt := req.ExpiresAt.UTC()
	if req.ExpiresAt.IsZero() {
		ttl := req.TTL
		if ttl <= 0 {
			ttl = DefaultCommandTTL
		}
		expiresAt = now.Add(ttl)
	}
	command := Command{CommandNo: uc.nextCommandNo(), DeviceNo: req.DeviceNo, Action: req.Action, CellNo: req.CellNo, Payload: append([]byte(nil), req.Payload...), IdempotencyKey: req.IdempotencyKey, Status: CommandPending, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now}
	created, err := uc.repo.CreateCommand(ctx, command)
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			if existing, findErr := uc.repo.FindCommandByIdempotencyKey(ctx, req.IdempotencyKey); findErr == nil {
				if !sameCommandRequest(existing, req) {
					return existing, ErrIdempotencyConflict
				}
				if existing.Status != CommandPending {
					return existing, nil
				}
				return uc.run(ctx, existing)
			}
		}
		return Command{}, err
	}
	if !now.Before(created.ExpiresAt) {
		return uc.expire(ctx, created)
	}
	return uc.run(ctx, created)
}

func (uc *UseCase) Send(ctx context.Context, req CommandRequest) (Command, error) {
	return uc.Execute(ctx, req)
}

func (uc *UseCase) run(ctx context.Context, command Command) (Command, error) {
	now := uc.now().UTC()
	if !now.Before(command.ExpiresAt) {
		return uc.expire(ctx, command)
	}
	if command.AttemptCount >= MaxCommandAttempts {
		return command, ErrMaxAttempts
	}
	device, err := uc.repo.FindDevice(ctx, command.DeviceNo)
	if err != nil {
		return command, err
	}
	if err := uc.validateDevice(device); err != nil {
		return command, err
	}
	marked, err := uc.repo.MarkRunning(ctx, command.CommandNo, now)
	if err != nil {
		return command, err
	}
	if !marked {
		if !now.Before(command.ExpiresAt) {
			return uc.expire(ctx, command)
		}
		if command.AttemptCount >= MaxCommandAttempts {
			return command, ErrMaxAttempts
		}
		return command, ErrCommandBusy
	}
	command.AttemptCount++
	result, gatewayErr := uc.gateway.Execute(ctx, GatewayCommand{DeviceNo: command.DeviceNo, Action: command.Action, CellNo: command.CellNo, Payload: command.Payload})
	status := CommandSucceeded
	errorCode := ""
	if gatewayErr != nil {
		if errors.Is(gatewayErr, ErrGatewayTimeout) || errors.Is(gatewayErr, context.DeadlineExceeded) || errors.Is(gatewayErr, context.Canceled) {
			status = CommandTimedOut
			errorCode = ErrorCodeCommandTimeout
			result.Retryable = true
		} else {
			status = CommandFailed
			errorCode = ErrorCodeCommandFailed
		}
	}
	if err := uc.completeTerminal(ctx, command.CommandNo, status, result, errorCode); err != nil {
		return command, err
	}
	command.Status, command.Result, command.ErrorCode = status, result, errorCode
	return command, gatewayErr
}

func (uc *UseCase) expire(ctx context.Context, command Command) (Command, error) {
	if err := uc.completeTerminal(ctx, command.CommandNo, CommandExpired, GatewayResult{}, ErrorCodeCommandExpired); err != nil {
		return command, err
	}
	command.Status = CommandExpired
	command.ErrorCode = ErrorCodeCommandExpired
	return command, ErrCommandExpired
}

func (uc *UseCase) completeTerminal(ctx context.Context, commandNo string, status CommandStatus, result GatewayResult, errorCode string) error {
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), terminalPersistenceTimeout)
	defer cancel()
	return uc.repo.CompleteCommand(persistCtx, commandNo, status, result, errorCode)
}

func (uc *UseCase) validateDevice(device Device) error {
	if device.NetworkStatus != NetworkOnline || device.LastHeartbeatAt.IsZero() || !uc.now().UTC().Before(device.LastHeartbeatAt.UTC().Add(uc.offlineThreshold)) {
		return ErrDeviceOffline
	}
	switch device.OperationalStatus {
	case OperationalActive:
		return nil
	case OperationalMaintenance:
		return ErrDeviceMaintenance
	case OperationalDisabled:
		return ErrDeviceDisabled
	default:
		return ErrInvalidDeviceState
	}
}

func (uc *UseCase) nextCommandNo() string {
	sequence := atomic.AddUint64(&uc.commandSequence, 1)
	// device_commands.command_no is CHAR(26); keep the generated identifier within
	// that bound while retaining monotonic timestamp/sequence uniqueness.
	return fmt.Sprintf("%020d%06x", uc.now().UTC().UnixNano(), sequence)
}

func validAction(action Action) bool {
	return action == ActionOpenDoor || action == ActionQueryStatus
}

func sameCommandRequest(command Command, request CommandRequest) bool {
	return command.DeviceNo == request.DeviceNo &&
		command.Action == request.Action &&
		command.CellNo == request.CellNo &&
		semanticallyEqualJSON(command.Payload, request.Payload)
}

func semanticallyEqualJSON(left, right []byte) bool {
	if len(left) == 0 {
		left = []byte(`{}`)
	}
	if len(right) == 0 {
		right = []byte(`{}`)
	}
	var leftValue, rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return bytes.Equal(left, right)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return bytes.Equal(left, right)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
