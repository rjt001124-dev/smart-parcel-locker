package data

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

var ErrNilDatabase = errors.New("locker database is required")

var errDataUnavailable = errors.New("locker data unavailable")

const defaultOfflineThreshold = 5 * time.Minute

type Repository struct {
	db               *sql.DB
	offlineThreshold time.Duration
	now              func() time.Time
}

var _ biz.Repository = (*Repository)(nil)

func NewRepository(db *sql.DB, offlineThreshold time.Duration, clock func() time.Time) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	if offlineThreshold <= 0 {
		offlineThreshold = defaultOfflineThreshold
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Repository{db: db, offlineThreshold: offlineThreshold, now: clock}, nil
}
func (r *Repository) FindReservation(ctx context.Context, key string) (biz.Reservation, error) {
	var x biz.Reservation
	var expiresAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT c.id,c.device_id,d.device_no,d.site_id,c.cell_no,c.size,c.occupancy_status,c.reservation_key,c.lock_expires_at FROM locker_cells c JOIN locker_devices d ON d.id=c.device_id WHERE c.reservation_key=?`, key).Scan(&x.CellID, &x.DeviceID, &x.DeviceNo, &x.SiteID, &x.CellNo, &x.Size, &x.Status, &x.ReservationKey, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return biz.Reservation{}, biz.ErrReservationNotFound
	}
	if err != nil {
		return biz.Reservation{}, sanitizeLockerError(ctx, err)
	}
	if expiresAt.Valid {
		x.ExpiresAt = expiresAt.Time.UTC()
	}
	return x, nil
}
func (r *Repository) ReserveAvailable(ctx context.Context, req biz.ReserveRequest, expiresAt time.Time) (biz.Reservation, error) {
	if existing, findErr := r.FindReservation(ctx, req.ReservationKey); findErr == nil {
		return existing, nil
	} else if !errors.Is(findErr, biz.ErrReservationNotFound) {
		return biz.Reservation{}, findErr
	}
	return r.reserveAvailable(ctx, req, expiresAt, true)
}

func (r *Repository) reserveAvailable(ctx context.Context, req biz.ReserveRequest, expiresAt time.Time, skipLocked bool) (biz.Reservation, error) {
	now := r.now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return biz.Reservation{}, sanitizeLockerError(ctx, err)
	}
	defer tx.Rollback()
	var x biz.Reservation
	if !skipLocked {
		var expiresAt sql.NullTime
		err = tx.QueryRowContext(ctx, `SELECT c.id,c.device_id,d.device_no,d.site_id,c.cell_no,c.size,c.occupancy_status,c.reservation_key,c.lock_expires_at FROM locker_cells c JOIN locker_devices d ON d.id=c.device_id WHERE c.reservation_key=? FOR UPDATE`, req.ReservationKey).Scan(&x.CellID, &x.DeviceID, &x.DeviceNo, &x.SiteID, &x.CellNo, &x.Size, &x.Status, &x.ReservationKey, &expiresAt)
		if err == nil {
			if expiresAt.Valid {
				x.ExpiresAt = expiresAt.Time.UTC()
			}
			_ = tx.Rollback()
			return x, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return biz.Reservation{}, sanitizeLockerError(ctx, err)
		}
	}
	lockClause := "FOR UPDATE SKIP LOCKED"
	if !skipLocked {
		lockClause = "FOR UPDATE"
	}
	query := `SELECT c.id,c.device_id,d.device_no,d.site_id,c.cell_no,c.size FROM locker_cells c JOIN locker_devices d ON d.id=c.device_id JOIN sites s ON s.id=d.site_id WHERE s.id=? AND s.service_status='ACTIVE' AND d.network_status='ONLINE' AND d.operational_status='ACTIVE' AND d.last_heartbeat_at>=? AND c.size=? AND c.current_order_id IS NULL AND (c.occupancy_status='IDLE' OR (c.occupancy_status='LOCKED' AND c.lock_expires_at<?)) ORDER BY c.id LIMIT 1 ` + lockClause
	err = tx.QueryRowContext(ctx, query, req.SiteID, now.Add(-r.offlineThreshold), req.Size, now).Scan(&x.CellID, &x.DeviceID, &x.DeviceNo, &x.SiteID, &x.CellNo, &x.Size)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		if existing, findErr := r.FindReservation(ctx, req.ReservationKey); findErr == nil {
			return existing, nil
		}
		if skipLocked {
			return r.reserveAvailable(ctx, req, expiresAt, false)
		}
		return biz.Reservation{}, biz.ErrCellNotAvailable
	}
	if err != nil {
		return biz.Reservation{}, sanitizeLockerError(ctx, err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE locker_cells SET occupancy_status='LOCKED', reservation_key=?, lock_expires_at=?, version=version+1 WHERE id=? AND current_order_id IS NULL AND (occupancy_status='IDLE' OR (occupancy_status='LOCKED' AND lock_expires_at<?))`, req.ReservationKey, expiresAt.UTC(), x.CellID, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			_ = tx.Rollback()
			if existing, e := r.FindReservation(ctx, req.ReservationKey); e == nil {
				return existing, nil
			}
		}
		return biz.Reservation{}, sanitizeLockerError(ctx, err)
	}
	if changed, rowsErr := result.RowsAffected(); rowsErr != nil || changed != 1 {
		return biz.Reservation{}, biz.ErrCellNotAvailable
	}
	if err = tx.Commit(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			_ = tx.Rollback()
			if existing, findErr := r.FindReservation(ctx, req.ReservationKey); findErr == nil {
				return existing, nil
			}
		}
		return biz.Reservation{}, sanitizeLockerError(ctx, err)
	}
	x.Status = biz.StatusLocked
	x.ReservationKey = req.ReservationKey
	x.ExpiresAt = expiresAt.UTC()
	return x, nil
}
func (r *Repository) Release(ctx context.Context, cellID uint64, key string) error {
	if cellID == 0 {
		return biz.ErrInvalidCellID
	}
	var existingKey sql.NullString
	var status biz.CellStatus
	var orderID sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `SELECT reservation_key, occupancy_status, current_order_id FROM locker_cells WHERE id=?`, cellID).Scan(&existingKey, &status, &orderID); errors.Is(err, sql.ErrNoRows) {
		return biz.ErrReservationNotFound
	} else if err != nil {
		return sanitizeLockerError(ctx, err)
	}
	if !existingKey.Valid || existingKey.String != key || status != biz.StatusLocked || orderID.Valid {
		return biz.ErrReservationKeyMismatch
	}
	res, err := r.db.ExecContext(ctx, `UPDATE locker_cells SET occupancy_status='IDLE', reservation_key=NULL, lock_expires_at=NULL, version=version+1 WHERE id=? AND reservation_key=? AND occupancy_status='LOCKED' AND current_order_id IS NULL`, cellID, key)
	if err != nil {
		return sanitizeLockerError(ctx, err)
	}
	n, rowsErr := res.RowsAffected()
	if rowsErr != nil {
		return sanitizeLockerError(ctx, rowsErr)
	}
	if n == 0 {
		return biz.ErrReservationKeyMismatch
	}
	return nil
}
func (r *Repository) ReclaimExpired(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, sanitizeLockerError(ctx, err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,reservation_key FROM locker_cells WHERE occupancy_status='LOCKED' AND current_order_id IS NULL AND lock_expires_at<? ORDER BY id LIMIT ? FOR UPDATE SKIP LOCKED`, now.UTC(), limit)
	if err != nil {
		return 0, sanitizeLockerError(ctx, err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id uint64
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			return 0, sanitizeLockerError(ctx, err)
		}
		res, err := tx.ExecContext(ctx, `UPDATE locker_cells SET occupancy_status='IDLE',reservation_key=NULL,lock_expires_at=NULL,version=version+1 WHERE id=? AND reservation_key=? AND occupancy_status='LOCKED' AND current_order_id IS NULL`, id, key)
		if err != nil {
			return 0, sanitizeLockerError(ctx, err)
		}
		c, _ := res.RowsAffected()
		n += int(c)
	}
	if err := rows.Err(); err != nil {
		return 0, sanitizeLockerError(ctx, err)
	}
	if err := rows.Close(); err != nil {
		return 0, sanitizeLockerError(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, sanitizeLockerError(ctx, err)
	}
	return n, nil
}
func sanitizeLockerError(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errDataUnavailable
}
