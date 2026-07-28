package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/order/biz"
)

var (
	ErrNilDatabase     = errors.New("order database is required")
	errDataUnavailable = errors.New("order data unavailable")
)

type Repository struct {
	db  *sql.DB
	now func() time.Time
}

var _ biz.Repository = (*Repository)(nil)

func NewRepository(db *sql.DB, clock func() time.Time) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Repository{db: db, now: clock}, nil
}

const orderColumns = `o.id, o.order_no, o.user_id, o.site_id, s.name, o.cell_id, c.cell_no, o.size, o.status, o.duration_minutes, o.rent_fee_fen, o.deposit_fen, o.discount_fen, o.total_fen, o.overdue_fee_fen, o.paid_at, o.deposited_at, o.expires_at, o.completed_at, o.created_at, o.updated_at`

const orderFrom = `FROM orders o
	JOIN sites s ON s.id = o.site_id
	JOIN locker_cells c ON c.id = o.cell_id `

func scanOrder(scan func(...any) error) (*biz.Order, error) {
	var o biz.Order
	var siteName, cellNo, size, status string
	var paidAt, depositedAt, expiresAt, completedAt sql.NullTime
	if err := scan(
		&o.ID, &o.OrderNo, &o.UserID, &o.SiteID, &siteName, &o.CellID, &cellNo,
		&size, &status, &o.DurationMinutes,
		&o.RentFeeFen, &o.DepositFen, &o.DiscountFen, &o.TotalFen, &o.OverdueFeeFen,
		&paidAt, &depositedAt, &expiresAt, &completedAt, &o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	o.SiteName = siteName
	o.CellNo = cellNo
	o.Size = biz.OrderSize(size)
	o.Status = biz.OrderStatus(status)
	o.PaidAt = nullTimePtr(paidAt)
	o.DepositedAt = nullTimePtr(depositedAt)
	o.ExpiresAt = nullTimePtr(expiresAt)
	o.CompletedAt = nullTimePtr(completedAt)
	return &o, nil
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.UTC()
	return &t
}

func (r *Repository) CreateOrder(ctx context.Context, order *biz.Order) (*biz.Order, error) {
	now := r.now().UTC()
	var siteExists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM sites WHERE id = ?`, order.SiteID).Scan(&siteExists); errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrSiteNotFound
	} else if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	defer tx.Rollback()

	var cellID uint64
	var cellNo string
	cellQuery := `SELECT c.id, c.cell_no
		FROM locker_cells c
		JOIN locker_devices d ON d.id = c.device_id
		WHERE d.site_id = ? AND c.size = ? AND c.occupancy_status = 'IDLE' AND c.current_order_id IS NULL
		ORDER BY c.id
		LIMIT 1 FOR UPDATE SKIP LOCKED`
	if err := tx.QueryRowContext(ctx, cellQuery, order.SiteID, string(order.Size)).Scan(&cellID, &cellNo); errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrNoAvailableCell
	} else if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	result, err := tx.ExecContext(ctx, `INSERT INTO orders
		(order_no, user_id, site_id, cell_id, size, status, duration_minutes, rent_fee_fen, deposit_fen, discount_fen, total_fen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.OrderNo, order.UserID, order.SiteID, cellID, string(order.Size), string(order.Status),
		order.DurationMinutes, order.RentFeeFen, order.DepositFen, order.DiscountFen, order.TotalFen,
		now, now)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	cellResult, err := tx.ExecContext(ctx, `UPDATE locker_cells
		SET occupancy_status = 'LOCKED', current_order_id = ?, version = version + 1
		WHERE id = ? AND occupancy_status = 'IDLE' AND current_order_id IS NULL`, id, cellID)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	if affected, aerr := cellResult.RowsAffected(); aerr != nil || affected != 1 {
		return nil, biz.ErrNoAvailableCell
	}

	if err := r.appendLog(ctx, tx, uint64(id), "", string(order.Status), "CREATE", "system", now); err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	order.ID = uint64(id)
	order.CellID = cellID
	order.CellNo = cellNo
	order.CreatedAt = now
	order.UpdatedAt = now
	return order, nil
}

func (r *Repository) GetOrder(ctx context.Context, orderID uint64) (*biz.Order, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+orderColumns+` `+orderFrom+` WHERE o.id = ?`, orderID)
	order, err := scanOrder(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrOrderNotFound
	}
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	return order, nil
}

func (r *Repository) ListOrders(ctx context.Context, userID string) ([]*biz.Order, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+orderColumns+` `+orderFrom+` WHERE o.user_id = ? ORDER BY o.created_at DESC, o.id DESC`, userID)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	defer rows.Close()
	orders := make([]*biz.Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows.Scan)
		if err != nil {
			return nil, sanitizeOrderError(ctx, err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	return orders, nil
}

func (r *Repository) MarkOverdue(ctx context.Context, orderID uint64, overdueFeeFen int64) error {
	now := r.now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return sanitizeOrderError(ctx, err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'OVERDUE', overdue_fee_fen = ?, updated_at = ? WHERE id = ? AND status = 'IN_STORAGE'`, overdueFeeFen, now, orderID)
	if err != nil {
		return sanitizeOrderError(ctx, err)
	}
	if affected, aerr := result.RowsAffected(); aerr != nil {
		return sanitizeOrderError(ctx, aerr)
	} else if affected == 0 {
		// Already left IN_STORAGE (idempotent) or lost a race; nothing to do.
		return tx.Commit()
	}
	if err := r.appendLog(ctx, tx, orderID, string(biz.StatusInStorage), string(biz.StatusOverdue), "OVERDUE_LAZY", "system", now); err != nil {
		return sanitizeOrderError(ctx, err)
	}
	return tx.Commit()
}

func (r *Repository) CancelOrder(ctx context.Context, orderID uint64) (*biz.Order, error) {
	return r.transition(ctx, orderID, func(tx *sql.Tx, order *biz.Order, now time.Time) error {
		if order.Status != biz.StatusPendingPayment {
			return biz.ErrOrderStateConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'CANCELLED', updated_at = ? WHERE id = ? AND status = 'PENDING_PAYMENT'`, now, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		if err := r.releaseCell(ctx, tx, order.CellID, orderID); err != nil {
			return err
		}
		return r.appendLog(ctx, tx, orderID, string(order.Status), string(biz.StatusCancelled), "CANCEL", "user", now)
	}, string(biz.StatusCancelled))
}

func (r *Repository) OpenDepositDoor(ctx context.Context, orderID uint64) (*biz.Order, error) {
	return r.transition(ctx, orderID, func(tx *sql.Tx, order *biz.Order, now time.Time) error {
		if order.Status != biz.StatusAwaitingDeposit {
			return biz.ErrOrderStateConflict
		}
		expiresAt := now.Add(time.Duration(order.DurationMinutes) * time.Minute)
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'IN_STORAGE', deposited_at = ?, expires_at = ?, updated_at = ? WHERE id = ? AND status = 'AWAITING_DEPOSIT'`, now, expiresAt, now, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE locker_cells SET occupancy_status = 'OCCUPIED', version = version + 1 WHERE id = ? AND current_order_id = ?`, order.CellID, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		return r.appendLog(ctx, tx, orderID, string(order.Status), string(biz.StatusInStorage), "DEPOSIT", "user", now)
	}, string(biz.StatusInStorage))
}

func (r *Repository) OpenPickupDoor(ctx context.Context, orderID uint64) (*biz.Order, error) {
	return r.transition(ctx, orderID, func(tx *sql.Tx, order *biz.Order, now time.Time) error {
		switch order.Status {
		case biz.StatusAwaitingPickup:
			// valid
		case biz.StatusInStorage:
			if order.ExpiresAt != nil && now.After(*order.ExpiresAt) {
				return biz.ErrOrderStateConflict
			}
		default:
			return biz.ErrOrderStateConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'COMPLETED', completed_at = ?, updated_at = ? WHERE id = ? AND (status = 'IN_STORAGE' OR status = 'AWAITING_PICKUP')`, now, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		if err := r.releaseCell(ctx, tx, order.CellID, orderID); err != nil {
			return err
		}
		return r.appendLog(ctx, tx, orderID, string(order.Status), string(biz.StatusCompleted), "PICKUP", "user", now)
	}, string(biz.StatusCompleted))
}

func (r *Repository) ConfirmDevelopmentPayment(ctx context.Context, orderID uint64) (*biz.Order, error) {
	return r.transition(ctx, orderID, func(tx *sql.Tx, order *biz.Order, now time.Time) error {
		if order.Status != biz.StatusPendingPayment {
			return biz.ErrOrderStateConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'AWAITING_DEPOSIT', paid_at = ?, updated_at = ? WHERE id = ? AND status = 'PENDING_PAYMENT'`, now, now, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		// Two consecutive status logs: PENDING_PAYMENT -> PAID -> AWAITING_DEPOSIT.
		if err := r.appendLog(ctx, tx, orderID, string(biz.StatusPendingPayment), string(biz.StatusPaid), "DEV_PAYMENT", "dev-payment", now); err != nil {
			return err
		}
		return r.appendLog(ctx, tx, orderID, string(biz.StatusPaid), string(biz.StatusAwaitingDeposit), "DEV_PAYMENT", "dev-payment", now)
	}, string(biz.StatusAwaitingDeposit))
}

func (r *Repository) SettleOverdue(ctx context.Context, orderID uint64) (*biz.Order, error) {
	return r.transition(ctx, orderID, func(tx *sql.Tx, order *biz.Order, now time.Time) error {
		if order.Status != biz.StatusOverdue {
			return biz.ErrOrderStateConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = 'AWAITING_PICKUP', overdue_fee_fen = 0, updated_at = ? WHERE id = ? AND status = 'OVERDUE'`, now, orderID); err != nil {
			return sanitizeOrderError(ctx, err)
		}
		return r.appendLog(ctx, tx, orderID, string(biz.StatusOverdue), string(biz.StatusAwaitingPickup), "SETTLE_OVERDUE", "dev-overdue", now)
	}, string(biz.StatusAwaitingPickup))
}

// transition loads the order row with a row lock, runs the mutation callback,
// and returns the freshly reloaded order. The callback must validate the
// source status and return ErrOrderStateConflict for illegal transitions.
func (r *Repository) transition(ctx context.Context, orderID uint64, mutate func(*sql.Tx, *biz.Order, time.Time) error, _ string) (*biz.Order, error) {
	now := r.now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+orderColumns+` `+orderFrom+` WHERE o.id = ? FOR UPDATE`, orderID)
	order, err := scanOrder(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrOrderNotFound
	}
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}

	if err := mutate(tx, order, now); err != nil {
		return nil, err
	}

	row = tx.QueryRowContext(ctx, `SELECT `+orderColumns+` `+orderFrom+` WHERE o.id = ?`, orderID)
	reloaded, err := scanOrder(row.Scan)
	if err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, sanitizeOrderError(ctx, err)
	}
	return reloaded, nil
}

func (r *Repository) appendLog(ctx context.Context, tx *sql.Tx, orderID uint64, from, to, reason, actor string, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO order_status_logs (order_id, from_status, to_status, reason, actor, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		orderID, from, to, reason, actor, now); err != nil {
		return sanitizeOrderError(ctx, err)
	}
	return nil
}

func (r *Repository) releaseCell(ctx context.Context, tx *sql.Tx, cellID, orderID uint64) error {
	if _, err := tx.ExecContext(ctx, `UPDATE locker_cells
		SET occupancy_status = 'IDLE', current_order_id = NULL, reservation_key = NULL, lock_expires_at = NULL, version = version + 1
		WHERE id = ? AND current_order_id = ?`, cellID, orderID); err != nil {
		return sanitizeOrderError(ctx, err)
	}
	return nil
}

func sanitizeOrderError(ctx context.Context, err error) error {
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
