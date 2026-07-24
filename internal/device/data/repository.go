package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	devicebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

var (
	ErrNilDatabase       = errors.New("device database is required")
	ErrInvalidPayload    = errors.New("invalid command payload")
	ErrInvalidCommand    = errors.New("invalid command status")
	ErrCommandNotRunning = errors.New("command is not running")
	errDataUnavailable   = errors.New("device data unavailable")
)

// Repository persists device heartbeats and gateway commands in MySQL.
type Repository struct {
	db  *sql.DB
	now func() time.Time
}

var _ devicebiz.Repository = (*Repository)(nil)

// NewRepository constructs a repository. The optional clock is used only for
// defaults and is normalized to UTC; MySQL remains the source of persisted state.
func NewRepository(db *sql.DB, clocks ...func() time.Time) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	clock := func() time.Time { return time.Now().UTC() }
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &Repository{db: db, now: clock}, nil
}

func (r *Repository) UpdateHeartbeat(ctx context.Context, deviceNo string, heartbeatAt time.Time, online bool) error {
	return r.RecordHeartbeat(ctx, devicebiz.DeviceHeartbeat{DeviceNo: deviceNo, ReportedAt: heartbeatAt, Online: online})
}

func (r *Repository) RecordHeartbeat(ctx context.Context, heartbeat devicebiz.DeviceHeartbeat) error {
	status := devicebiz.NetworkOffline
	if heartbeat.Online {
		status = devicebiz.NetworkOnline
	}
	var result sql.Result
	var err error
	if heartbeat.FirmwareVersion == "" {
		result, err = r.db.ExecContext(ctx, `
		UPDATE locker_devices
		SET network_status = ?, last_heartbeat_at = ?
		WHERE device_no = ?`, status, heartbeat.ReportedAt.UTC(), heartbeat.DeviceNo)
	} else {
		result, err = r.db.ExecContext(ctx, `
		UPDATE locker_devices
		SET network_status = ?, last_heartbeat_at = ?, firmware_version = ?
		WHERE device_no = ?`, status, heartbeat.ReportedAt.UTC(), heartbeat.FirmwareVersion, heartbeat.DeviceNo)
	}
	if err != nil {
		return sanitizeDataError(ctx, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return sanitizeDataError(ctx, err)
	}
	if n == 0 {
		var exists int
		err := r.db.QueryRowContext(ctx, `SELECT 1 FROM locker_devices WHERE device_no = ?`, heartbeat.DeviceNo).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return devicebiz.ErrDeviceNotFound
		}
		if err != nil {
			return sanitizeDataError(ctx, err)
		}
	}
	return nil
}

func (r *Repository) FindDevice(ctx context.Context, deviceNo string) (devicebiz.Device, error) {
	var d devicebiz.Device
	var heartbeat sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, device_no, network_status, operational_status, last_heartbeat_at
		FROM locker_devices WHERE device_no = ?`, deviceNo).
		Scan(&d.ID, &d.DeviceNo, &d.NetworkStatus, &d.OperationalStatus, &heartbeat)
	if errors.Is(err, sql.ErrNoRows) {
		return devicebiz.Device{}, devicebiz.ErrDeviceNotFound
	}
	if err != nil {
		return devicebiz.Device{}, sanitizeDataError(ctx, err)
	}
	if heartbeat.Valid {
		d.LastHeartbeatAt = heartbeat.Time.UTC()
	}
	return d, nil
}

func (r *Repository) FindCommandByIdempotencyKey(ctx context.Context, key string) (devicebiz.Command, error) {
	return r.findCommand(ctx, `c.idempotency_key = ?`, key)
}

func (r *Repository) FindCommand(ctx context.Context, commandNo string) (devicebiz.Command, error) {
	return r.findCommand(ctx, `c.command_no = ?`, commandNo)
}

func (r *Repository) findCommand(ctx context.Context, predicate string, value string) (devicebiz.Command, error) {
	var c devicebiz.Command
	var payload, result []byte
	var resultJSON, errorCode sql.NullString
	var created, updated time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.command_no, d.device_no, c.action, c.payload_json,
		       c.cell_no, c.idempotency_key, c.status, c.expires_at, c.attempt_count,
		       c.result_json, c.error_code, c.created_at, c.updated_at
		FROM device_commands c
		JOIN locker_devices d ON d.id = c.device_id
		WHERE `+predicate, value).
		Scan(&c.ID, &c.CommandNo, &c.DeviceNo, &c.Action, &payload,
			&c.CellNo, &c.IdempotencyKey, &c.Status, &c.ExpiresAt, &c.AttemptCount,
			&resultJSON, &errorCode, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return devicebiz.Command{}, devicebiz.ErrCommandNotFound
	}
	if err != nil {
		return devicebiz.Command{}, sanitizeDataError(ctx, err)
	}
	if err := decodeJSON(payload, &c.Payload); err != nil {
		return devicebiz.Command{}, err
	}
	if resultJSON.Valid && resultJSON.String != "" {
		result = []byte(resultJSON.String)
		if err := decodeResult(result, &c.Result); err != nil {
			return devicebiz.Command{}, err
		}
	}
	c.ExpiresAt = c.ExpiresAt.UTC()
	c.CreatedAt, c.UpdatedAt = created.UTC(), updated.UTC()
	if errorCode.Valid {
		c.ErrorCode = errorCode.String
	}
	return c, nil
}

func (r *Repository) CreateCommand(ctx context.Context, command devicebiz.Command) (devicebiz.Command, error) {
	payload, err := normalizePayload(command.Payload)
	if err != nil {
		return devicebiz.Command{}, err
	}
	if command.Status == "" {
		command.Status = devicebiz.CommandPending
	}
	if command.Status != devicebiz.CommandPending {
		return devicebiz.Command{}, ErrInvalidCommand
	}
	if command.Action != devicebiz.ActionOpenDoor && command.Action != devicebiz.ActionQueryStatus {
		return devicebiz.Command{}, devicebiz.ErrInvalidAction
	}
	if command.ExpiresAt.IsZero() {
		command.ExpiresAt = r.now().UTC().Add(devicebiz.DefaultCommandTTL)
	} else {
		command.ExpiresAt = command.ExpiresAt.UTC()
	}
	if command.CreatedAt.IsZero() {
		command.CreatedAt = r.now().UTC()
	} else {
		command.CreatedAt = command.CreatedAt.UTC()
	}
	if command.UpdatedAt.IsZero() {
		command.UpdatedAt = command.CreatedAt
	} else {
		command.UpdatedAt = command.UpdatedAt.UTC()
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO device_commands
		(command_no, device_id, cell_no, action, payload_json, idempotency_key, status,
		 expires_at, attempt_count, result_json, error_code, created_at, updated_at)
		SELECT ?, id, ?, ?, ?, ?, 'PENDING', ?, 0, NULL, NULL, ?, ?
		FROM locker_devices WHERE device_no = ?`,
		command.CommandNo, command.CellNo, command.Action, payload, command.IdempotencyKey,
		command.ExpiresAt, command.CreatedAt, command.UpdatedAt, command.DeviceNo)
	if err != nil {
		if isDuplicate(err) {
			return devicebiz.Command{}, devicebiz.ErrIdempotencyConflict
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return devicebiz.Command{}, ctxErr
		}
		return devicebiz.Command{}, errDataUnavailable
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return devicebiz.Command{}, devicebiz.ErrDeviceNotFound
	}
	created, err := r.FindCommandByIdempotencyKey(ctx, command.IdempotencyKey)
	if err != nil {
		return devicebiz.Command{}, err
	}
	return created, nil
}

func (r *Repository) MarkRunning(ctx context.Context, commandNo string, now time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE device_commands
		SET status = 'RUNNING', attempt_count = attempt_count + 1
		WHERE command_no = ? AND status = 'PENDING'
		  AND expires_at > ? AND attempt_count < ?`, commandNo, now.UTC(), devicebiz.MaxCommandAttempts)
	if err != nil {
		return false, sanitizeDataError(ctx, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, sanitizeDataError(ctx, err)
	}
	return n == 1, nil
}

func (r *Repository) CompleteCommand(ctx context.Context, commandNo string, status devicebiz.CommandStatus, result devicebiz.GatewayResult, errorCode string) error {
	if status != devicebiz.CommandSucceeded && status != devicebiz.CommandFailed && status != devicebiz.CommandTimedOut && status != devicebiz.CommandExpired {
		return ErrInvalidCommand
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return ErrInvalidPayload
	}
	var query string
	var args []any
	if status == devicebiz.CommandExpired {
		query = `UPDATE device_commands SET status=?, result_json=?, error_code=? WHERE command_no=? AND status IN ('PENDING','RUNNING')`
		args = []any{status, resultJSON, nullString(errorCode), commandNo}
	} else {
		query = `UPDATE device_commands SET status=?, result_json=?, error_code=? WHERE command_no=? AND status='RUNNING'`
		args = []any{status, resultJSON, nullString(errorCode), commandNo}
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return sanitizeDataError(ctx, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return sanitizeDataError(ctx, err)
	}
	if n == 1 {
		return nil
	}
	return ErrCommandNotRunning
}

func normalizePayload(payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return []byte(`{}`), nil
	}
	if !json.Valid(payload) {
		return nil, ErrInvalidPayload
	}
	return append([]byte(nil), payload...), nil
}

func decodeJSON(raw []byte, target *[]byte) error {
	if len(raw) == 0 || !json.Valid(raw) {
		return ErrInvalidPayload
	}
	*target = append((*target)[:0], raw...)
	return nil
}

func decodeResult(raw []byte, target *devicebiz.GatewayResult) error {
	if !json.Valid(raw) {
		return ErrInvalidPayload
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return ErrInvalidPayload
	}
	return nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func sanitizeDataError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errDataUnavailable
}
