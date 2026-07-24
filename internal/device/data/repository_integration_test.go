//go:build integration

package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	devicebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

func TestRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil, nil); !errors.Is(err, ErrNilDatabase) {
		t.Fatal(err)
	}
}

func openDeviceIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is unset")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("mysql unavailable: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRepositoryDeviceCommandRoundTrip(t *testing.T) {
	db := openDeviceIntegrationDB(t)
	repo, err := NewRepository(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	prefix := fmt.Sprintf("T10-%d-", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM device_commands WHERE idempotency_key LIKE ?`, prefix+"%")
	})

	t.Run("heartbeat only addresses target and maps missing", func(t *testing.T) {
		before, err := repo.FindDevice(ctx, "DEV-SH-002")
		if err != nil {
			t.Fatal(err)
		}
		at := time.Now().UTC().Add(-time.Second)
		if err := repo.UpdateHeartbeat(ctx, "DEV-SH-001", at, false); err != nil {
			t.Fatal(err)
		}
		after, err := repo.FindDevice(ctx, "DEV-SH-002")
		if err != nil || after.NetworkStatus != before.NetworkStatus || !after.LastHeartbeatAt.Equal(before.LastHeartbeatAt) {
			t.Fatalf("non-target device changed: before=%+v after=%+v err=%v", before, after, err)
		}
		if err := repo.UpdateHeartbeat(ctx, "missing-device", at, true); !errors.Is(err, devicebiz.ErrDeviceNotFound) {
			t.Fatalf("missing heartbeat error=%v", err)
		}
		// Restore the seeded fixture for subsequent tests.
		if err := repo.UpdateHeartbeat(ctx, "DEV-SH-001", time.Now().UTC(), true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found and context errors are stable", func(t *testing.T) {
		if _, err := repo.FindDevice(ctx, "missing-device"); !errors.Is(err, devicebiz.ErrDeviceNotFound) {
			t.Fatalf("FindDevice missing error=%v", err)
		}
		if _, err := repo.FindCommandByIdempotencyKey(ctx, prefix+"missing"); !errors.Is(err, devicebiz.ErrCommandNotFound) {
			t.Fatalf("FindCommand missing error=%v", err)
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := repo.FindDevice(canceled, "DEV-SH-001"); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled context error=%v", err)
		}
		deadline, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer cancel()
		if _, err := repo.FindCommandByIdempotencyKey(deadline, prefix+"missing"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("deadline context error=%v", err)
		}
	})

	now := time.Now().UTC()
	newCommand := func(t *testing.T, suffix string, expires time.Time) devicebiz.Command {
		t.Helper()
		return devicebiz.Command{
			CommandNo:      fmt.Sprintf("%020d%06d", time.Now().UnixNano(), len(suffix)),
			DeviceNo:       "DEV-SH-001",
			Action:         devicebiz.ActionOpenDoor,
			Payload:        []byte(`{"secret":"payload-value","n":1}`),
			IdempotencyKey: prefix + suffix,
			Status:         devicebiz.CommandPending,
			ExpiresAt:      expires,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	t.Run("create idempotency and JSON round trip", func(t *testing.T) {
		command := newCommand(t, "roundtrip", now.Add(time.Minute))
		created, err := repo.CreateCommand(ctx, command)
		if err != nil {
			t.Fatal(err)
		}
		if string(created.Payload) != string(command.Payload) || created.Status != devicebiz.CommandPending {
			t.Fatalf("created=%+v", created)
		}
		found, err := repo.FindCommandByIdempotencyKey(ctx, command.IdempotencyKey)
		if err != nil || string(found.Payload) != string(command.Payload) {
			t.Fatalf("found=%+v err=%v", found, err)
		}
		if err := repo.CompleteCommand(ctx, command.CommandNo, devicebiz.CommandSucceeded, devicebiz.GatewayResult{Opened: true, DoorClosed: true}, ""); !errors.Is(err, ErrCommandNotRunning) {
			t.Fatalf("complete pending error=%v", err)
		}
		duplicate := command
		duplicate.CommandNo = fmt.Sprintf("%020d%06d", time.Now().UnixNano(), 2)
		if _, err := repo.CreateCommand(ctx, duplicate); !errors.Is(err, devicebiz.ErrIdempotencyConflict) {
			t.Fatalf("duplicate error=%v", err)
		}
		if ok, err := repo.MarkRunning(ctx, command.CommandNo, now); err != nil || !ok {
			t.Fatalf("mark running=%v err=%v", ok, err)
		}
		if err := repo.CompleteCommand(ctx, command.CommandNo, devicebiz.CommandSucceeded, devicebiz.GatewayResult{Opened: true, DoorClosed: true}, ""); err != nil {
			t.Fatal(err)
		}
		found, err = repo.FindCommandByIdempotencyKey(ctx, command.IdempotencyKey)
		if err != nil || !found.Result.Opened || !found.Result.DoorClosed {
			t.Fatalf("result round trip=%+v err=%v", found.Result, err)
		}
	})

	t.Run("conditional running and late completion", func(t *testing.T) {
		expired := newCommand(t, "expired", now.Add(-time.Second))
		if _, err := repo.CreateCommand(ctx, expired); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.MarkRunning(ctx, expired.CommandNo, now); err != nil || ok {
			t.Fatalf("expired mark=%v err=%v", ok, err)
		}
		if err := repo.CompleteCommand(ctx, expired.CommandNo, devicebiz.CommandExpired, devicebiz.GatewayResult{}, devicebiz.ErrorCodeCommandExpired); err != nil {
			t.Fatal(err)
		}
		if err := repo.CompleteCommand(ctx, expired.CommandNo, devicebiz.CommandSucceeded, devicebiz.GatewayResult{Opened: true}, ""); !errors.Is(err, ErrCommandNotRunning) {
			t.Fatalf("late expired result=%v", err)
		}

		running := newCommand(t, "running", now.Add(time.Minute))
		if _, err := repo.CreateCommand(ctx, running); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.MarkRunning(ctx, running.CommandNo, now); err != nil || !ok {
			t.Fatalf("running mark=%v err=%v", ok, err)
		}
		if err := repo.CompleteCommand(ctx, running.CommandNo, devicebiz.CommandTimedOut, devicebiz.GatewayResult{Retryable: true}, devicebiz.ErrorCodeCommandTimeout); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.MarkRunning(ctx, running.CommandNo, now); err != nil || ok {
			t.Fatalf("terminal mark=%v err=%v", ok, err)
		}
		if err := repo.CompleteCommand(ctx, running.CommandNo, devicebiz.CommandSucceeded, devicebiz.GatewayResult{Opened: true}, ""); !errors.Is(err, ErrCommandNotRunning) {
			t.Fatalf("late timed-out result=%v", err)
		}
	})

	t.Run("invalid payload errors are safe", func(t *testing.T) {
		bad := newCommand(t, "bad-payload", now.Add(time.Minute))
		bad.Payload = []byte(`{"secret":"raw-payload",`)
		err := func() error { _, err := repo.CreateCommand(ctx, bad); return err }()
		if !errors.Is(err, ErrInvalidPayload) || strings.Contains(err.Error(), "raw-payload") || strings.Contains(err.Error(), os.Getenv("TEST_MYSQL_DSN")) {
			t.Fatalf("unsafe invalid payload error=%v", err)
		}
	})
}
