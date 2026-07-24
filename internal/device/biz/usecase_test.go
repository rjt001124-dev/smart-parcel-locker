package biz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeRepository struct {
	mu          sync.Mutex
	device      Device
	commands    map[string]Command
	createCalls int
	markCalls   int
	complete    []CommandStatus
}

func (r *fakeRepository) UpdateHeartbeat(context.Context, string, time.Time, bool) error { return nil }

func (r *fakeRepository) FindDevice(context.Context, string) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.DeviceNo == "" {
		return Device{}, ErrDeviceNotFound
	}
	return r.device, nil
}

func (r *fakeRepository) FindCommandByIdempotencyKey(_ context.Context, key string) (Command, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	command, ok := r.commands[key]
	if !ok {
		return Command{}, ErrCommandNotFound
	}
	return command, nil
}

func (r *fakeRepository) CreateCommand(_ context.Context, command Command) (Command, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	if existing, ok := r.commands[command.IdempotencyKey]; ok {
		return existing, ErrIdempotencyConflict
	}
	if r.commands == nil {
		r.commands = make(map[string]Command)
	}
	r.commands[command.IdempotencyKey] = command
	return command, nil
}

func (r *fakeRepository) MarkRunning(_ context.Context, commandNo string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.markCalls++
	for key, command := range r.commands {
		if command.CommandNo != commandNo {
			continue
		}
		if !now.Before(command.ExpiresAt) || command.AttemptCount >= MaxCommandAttempts || command.Status != CommandPending {
			return false, nil
		}
		command.Status = CommandRunning
		command.AttemptCount++
		r.commands[key] = command
		return true, nil
	}
	return false, ErrCommandNotFound
}

func (r *fakeRepository) CompleteCommand(_ context.Context, commandNo string, status CommandStatus, result GatewayResult, errorCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, command := range r.commands {
		if command.CommandNo == commandNo {
			command.Status = status
			command.Result = result
			command.ErrorCode = errorCode
			r.commands[key] = command
			r.complete = append(r.complete, status)
			return nil
		}
	}
	return ErrCommandNotFound
}

type fakeGateway struct {
	mu     sync.Mutex
	calls  int
	result GatewayResult
	err    error
}

func (g *fakeGateway) Execute(context.Context, GatewayCommand) (GatewayResult, error) {
	g.mu.Lock()
	g.calls++
	g.mu.Unlock()
	return g.result, g.err
}

func fixedUseCase(repo Repository, gateway Gateway, now time.Time) *UseCase {
	return NewUseCase(repo, gateway, func() time.Time { return now }, time.Minute)
}

func TestUseCaseRejectsStaleHeartbeatBeforeGateway(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now.Add(-2 * time.Minute)}}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-stale"})
	if !errors.Is(err, ErrDeviceOffline) {
		t.Fatalf("error = %v, want ErrDeviceOffline", err)
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gateway.calls)
	}
}

func TestUseCaseRejectsExpiredCommandBeforeMarkRunning(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}}
	gateway := &fakeGateway{}
	uc := fixedUseCase(repo, gateway, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-expired", ExpiresAt: now.Add(-time.Second)})
	if !errors.Is(err, ErrCommandExpired) {
		t.Fatalf("error = %v, want ErrCommandExpired", err)
	}
	if repo.markCalls != 0 || gateway.calls != 0 {
		t.Fatalf("mark calls = %d, gateway calls = %d, want both 0", repo.markCalls, gateway.calls)
	}
}

func TestUseCaseReplaysIdempotentCommandWithoutGateway(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command)}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)
	req := CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-replay"}

	first, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.CommandNo != second.CommandNo || first.Status != second.Status || first.Result != second.Result {
		t.Fatalf("replay = %+v, first = %+v", second, first)
	}
	if gateway.calls != 1 {
		t.Fatalf("gateway calls = %d, want 1", gateway.calls)
	}
}

func TestUseCaseRejectsCommandAtMaximumAttempts(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: map[string]Command{
		"key-max": {CommandNo: "CMD-MAX", DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-max", Status: CommandPending, AttemptCount: MaxCommandAttempts, ExpiresAt: now.Add(time.Minute)},
	}}
	gateway := &fakeGateway{}
	uc := fixedUseCase(repo, gateway, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-max"})
	if !errors.Is(err, ErrMaxAttempts) {
		t.Fatalf("error = %v, want ErrMaxAttempts", err)
	}
	if gateway.calls != 0 || repo.markCalls != 0 {
		t.Fatalf("gateway calls = %d, mark calls = %d, want both 0", gateway.calls, repo.markCalls)
	}
}
