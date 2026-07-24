package biz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeRepository struct {
	mu             sync.Mutex
	device         Device
	commands       map[string]Command
	createCalls    int
	markCalls      int
	complete       []CommandStatus
	completeErr    error
	rejectCanceled bool
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

func (r *fakeRepository) CompleteCommand(ctx context.Context, commandNo string, status CommandStatus, result GatewayResult, errorCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rejectCanceled && ctx.Err() != nil {
		return ctx.Err()
	}
	if r.completeErr != nil {
		return r.completeErr
	}
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

func TestUseCasePropagatesExpiryPersistenceFailure(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	persistenceErr := errors.New("expiry persistence unavailable")
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command), completeErr: persistenceErr}
	uc := fixedUseCase(repo, &fakeGateway{}, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-expiry-persist", ExpiresAt: now.Add(-time.Second)})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("error = %v, want expiry persistence error", err)
	}
	if errors.Is(err, ErrCommandExpired) {
		t.Fatalf("error = %v, must not report expired when persistence failed", err)
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
	if first.AttemptCount != 1 || second.AttemptCount != 1 {
		t.Fatalf("attempt counts = first:%d second:%d, want both 1", first.AttemptCount, second.AttemptCount)
	}
	if persisted := repo.commands[req.IdempotencyKey].AttemptCount; persisted != 1 {
		t.Fatalf("persisted attempt count = %d, want 1", persisted)
	}
	if gateway.calls != 1 {
		t.Fatalf("gateway calls = %d, want 1", gateway.calls)
	}
}

func TestUseCaseReplaysSemanticallyEquivalentJSONPayload(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command)}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)

	firstReq := CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, CellNo: "A1", Payload: []byte(`{"door":{"open":true},"count":1}`), IdempotencyKey: "key-json-semantic"}
	first, err := uc.Execute(context.Background(), firstReq)
	if err != nil {
		t.Fatal(err)
	}
	replayReq := firstReq
	replayReq.Payload = []byte(`{ "count": 1, "door": { "open": true } }`)
	second, err := uc.Execute(context.Background(), replayReq)
	if err != nil {
		t.Fatalf("semantic replay error = %v", err)
	}
	if second.CommandNo != first.CommandNo || gateway.calls != 1 {
		t.Fatalf("semantic replay = %+v, first = %+v, gateway calls = %d", second, first, gateway.calls)
	}

	differentReq := replayReq
	differentReq.Payload = []byte(`{"count":2,"door":{"open":true}}`)
	if _, err := uc.Execute(context.Background(), differentReq); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different JSON payload error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestUseCaseRejectsIdempotencyKeyConflict(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: map[string]Command{
		"key-conflict": {CommandNo: "CMD-CONFLICT", DeviceNo: "DEV-001", Action: ActionOpenDoor, CellNo: "A1", Payload: []byte("payload-a"), IdempotencyKey: "key-conflict", Status: CommandSucceeded, ExpiresAt: now.Add(time.Minute)},
	}}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-002", Action: ActionOpenDoor, CellNo: "A1", Payload: []byte("payload-a"), IdempotencyKey: "key-conflict"})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gateway.calls)
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

func TestUseCaseReturnsIncrementedAttemptWhenCompletionFails(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	completionErr := errors.New("completion unavailable")
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command), completeErr: completionErr}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)

	command, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-attempt"})
	if !errors.Is(err, completionErr) {
		t.Fatalf("error = %v, want completion error", err)
	}
	if command.AttemptCount != 1 {
		t.Fatalf("returned attempt count = %d, want 1", command.AttemptCount)
	}
}

func TestUseCasePersistsStableTimeoutErrorCode(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command)}
	gateway := &fakeGateway{err: ErrGatewayTimeout}
	uc := fixedUseCase(repo, gateway, now)

	command, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-timeout"})
	if !errors.Is(err, ErrGatewayTimeout) {
		t.Fatalf("error = %v, want ErrGatewayTimeout", err)
	}
	if command.ErrorCode != "DEVICE_COMMAND_TIMEOUT" {
		t.Fatalf("error code = %q, want %q", command.ErrorCode, "DEVICE_COMMAND_TIMEOUT")
	}
	if !command.Result.Retryable {
		t.Fatal("timeout result Retryable = false, want true")
	}
}

func TestUseCasePersistsTimeoutAfterCallerCancellation(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command), rejectCanceled: true}
	gateway := &fakeGateway{err: ErrGatewayTimeout}
	uc := fixedUseCase(repo, gateway, now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	command, err := uc.Execute(ctx, CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-canceled-timeout"})
	if !errors.Is(err, ErrGatewayTimeout) {
		t.Fatalf("error = %v, want ErrGatewayTimeout", err)
	}
	if command.Status != CommandTimedOut || command.ErrorCode != ErrorCodeCommandTimeout {
		t.Fatalf("command = %+v, want timed-out terminal command", command)
	}
	persisted := repo.commands["key-canceled-timeout"]
	if persisted.Status != CommandTimedOut || persisted.ErrorCode != ErrorCodeCommandTimeout {
		t.Fatalf("persisted = %+v, want timed-out terminal command", persisted)
	}
}

func TestUseCasePersistsStableFailureErrorCode(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalActive, LastHeartbeatAt: now}, commands: make(map[string]Command)}
	gateway := &fakeGateway{err: ErrGatewayFailure}
	uc := fixedUseCase(repo, gateway, now)

	command, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-failure"})
	if !errors.Is(err, ErrGatewayFailure) {
		t.Fatalf("error = %v, want ErrGatewayFailure", err)
	}
	if command.ErrorCode != "DEVICE_COMMAND_FAILED" {
		t.Fatalf("error code = %q, want %q", command.ErrorCode, "DEVICE_COMMAND_FAILED")
	}
}

func TestUseCaseRejectsMaintenanceAndDisabledDevices(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		status OperationalStatus
		want   error
	}{
		{name: "maintenance", status: OperationalMaintenance, want: ErrDeviceMaintenance},
		{name: "disabled", status: OperationalDisabled, want: ErrDeviceDisabled},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: test.status, LastHeartbeatAt: now}, commands: make(map[string]Command)}
			gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
			uc := fixedUseCase(repo, gateway, now)

			_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-" + test.name})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if gateway.calls != 0 {
				t.Fatalf("gateway calls = %d, want 0", gateway.calls)
			}
		})
	}
}

func TestUseCaseRejectsUnknownOperationalStatus(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{device: Device{DeviceNo: "DEV-001", NetworkStatus: NetworkOnline, OperationalStatus: OperationalStatus("UNKNOWN"), LastHeartbeatAt: now}, commands: make(map[string]Command)}
	gateway := &fakeGateway{result: GatewayResult{Opened: true, DoorClosed: true}}
	uc := fixedUseCase(repo, gateway, now)

	_, err := uc.Execute(context.Background(), CommandRequest{DeviceNo: "DEV-001", Action: ActionOpenDoor, IdempotencyKey: "key-unknown-state"})
	if !errors.Is(err, ErrInvalidDeviceState) {
		t.Fatalf("error = %v, want ErrInvalidDeviceState", err)
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gateway.calls)
	}
}
