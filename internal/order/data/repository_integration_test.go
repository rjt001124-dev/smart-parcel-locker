//go:build integration

package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/order/biz"
)

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set; skipping MySQL integration test")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("open integration database failed")
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close integration database: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("integration database unavailable")
	}
	return db
}

func uniqueSuffix(prefix string) string {
	return prefix + fmt.Sprintf("%x", time.Now().UnixNano())
}

func insertFixtures(t *testing.T, db *sql.DB, now time.Time) (siteID, deviceID, cellID uint64) {
	t.Helper()
	ctx := context.Background()
	cityCode := uniqueSuffix("CITY-")
	if _, err := db.ExecContext(ctx, `INSERT INTO cities (code, name, province, enabled) VALUES (?, 'test', 'test', TRUE)`, cityCode); err != nil {
		t.Fatalf("insert city: %v", err)
	}
	t.Cleanup(func() { cleanupExec(t, db, "city", `DELETE FROM cities WHERE code = ?`, cityCode) })

	siteNo := uniqueSuffix("SITE-")
	res, err := db.ExecContext(ctx, `INSERT INTO sites (site_no, city_id, name, address, latitude, longitude, open_time, close_time, contact_phone, service_status) VALUES (?, (SELECT id FROM cities WHERE code = ?), 'test', 'test', 31.23, 121.47, '08:00:00', '22:00:00', 'test', 'ACTIVE')`, siteNo, cityCode)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	siteID, _ = res.LastInsertId()
	t.Cleanup(func() { cleanupExec(t, db, "site", `DELETE FROM sites WHERE site_no = ?`, siteNo) })

	deviceNo := uniqueSuffix("DEV-")
	res, err = db.ExecContext(ctx, `INSERT INTO locker_devices (device_no, site_id, network_status, operational_status, last_heartbeat_at) VALUES (?, ?, 'ONLINE', 'ACTIVE', ?)`, deviceNo, siteID, now)
	if err != nil {
		t.Fatalf("insert device: %v", err)
	}
	deviceID, _ = res.LastInsertId()
	t.Cleanup(func() { cleanupExec(t, db, "device", `DELETE FROM locker_devices WHERE device_no = ?`, deviceNo) })

	cellNo := uniqueSuffix("CELL-")
	res, err = db.ExecContext(ctx, `INSERT INTO locker_cells (device_id, cell_no, size, occupancy_status) VALUES (?, ?, 'SMALL', 'IDLE')`, deviceID, cellNo)
	if err != nil {
		t.Fatalf("insert cell: %v", err)
	}
	cellID, _ = res.LastInsertId()
	t.Cleanup(func() { cleanupExec(t, db, "cell", `DELETE FROM locker_cells WHERE cell_no = ?`, cellNo) })
	return siteID, deviceID, cellID
}

func cleanupExec(t *testing.T, db *sql.DB, label, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Errorf("cleanup %s: %v", label, err)
	}
}

func countLogs(t *testing.T, db *sql.DB, orderID uint64) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM order_status_logs WHERE order_id = ?`, orderID).Scan(&n); err != nil {
		t.Fatalf("count logs: %v", err)
	}
	return n
}

func cellState(t *testing.T, db *sql.DB, cellID uint64) (status string, orderID sql.NullInt64) {
	t.Helper()
	if err := db.QueryRowContext(context.Background(), `SELECT occupancy_status, current_order_id FROM locker_cells WHERE id = ?`, cellID).Scan(&status, &orderID); err != nil {
		t.Fatalf("read cell state: %v", err)
	}
	return status, orderID
}

func TestRepositoryLifecycleAgainstMySQL(t *testing.T) {
	db := openIntegrationDB(t)
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo, err := NewRepository(db, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	ctx := context.Background()
	siteID, _, cellID := insertFixtures(t, db, now)
	_ = cellID

	// CreateOrder locks an idle cell and starts PENDING_PAYMENT.
	order, err := repo.CreateOrder(ctx, &biz.Order{
		OrderNo:         uniqueSuffix("SPL-"),
		UserID:          biz.DefaultUserID,
		SiteID:          siteID,
		Size:            biz.SizeSmall,
		Status:          biz.StatusPendingPayment,
		DurationMinutes: 60,
		RentFeeFen:      300,
		TotalFen:        300,
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order.Status != biz.StatusPendingPayment {
		t.Fatalf("status = %s, want PENDING_PAYMENT", order.Status)
	}
	if countLogs(t, db, order.ID) != 1 {
		t.Fatalf("expected 1 status log after create, got %d", countLogs(t, db, order.ID))
	}
	if status, oid := cellState(t, db, order.CellID); status != "LOCKED" || !oid.Valid || oid.Int64 != int64(order.ID) {
		t.Fatalf("cell state = %s order=%v, want LOCKED bound to order %d", status, oid, order.ID)
	}

	// No available LARGE cell at this site must be reported.
	if _, err := repo.CreateOrder(ctx, &biz.Order{
		OrderNo: uniqueSuffix("SPL-"), UserID: biz.DefaultUserID, SiteID: siteID,
		Size: biz.SizeLarge, Status: biz.StatusPendingPayment, DurationMinutes: 60,
		RentFeeFen: 800, TotalFen: 800,
	}); !errors.Is(err, biz.ErrNoAvailableCell) {
		t.Fatalf("CreateOrder(LARGE) error = %v, want ErrNoAvailableCell", err)
	}

	// ConfirmDevelopmentPayment -> AWAITING_DEPOSIT with two extra logs.
	paid, err := repo.ConfirmDevelopmentPayment(ctx, order.ID)
	if err != nil {
		t.Fatalf("ConfirmDevelopmentPayment() error = %v", err)
	}
	if paid.Status != biz.StatusAwaitingDeposit || paid.PaidAt == nil {
		t.Fatalf("after payment status=%s paidAt=%v", paid.Status, paid.PaidAt)
	}
	if countLogs(t, db, order.ID) != 3 {
		t.Fatalf("expected 3 status logs after dev payment, got %d", countLogs(t, db, order.ID))
	}

	// Illegal dev payment retry must be rejected.
	if _, err := repo.ConfirmDevelopmentPayment(ctx, order.ID); !errors.Is(err, biz.ErrOrderStateConflict) {
		t.Fatalf("repeat ConfirmDevelopmentPayment error = %v, want ErrOrderStateConflict", err)
	}

	// OpenDepositDoor -> IN_STORAGE and occupies the cell.
	deposited, err := repo.OpenDepositDoor(ctx, order.ID)
	if err != nil {
		t.Fatalf("OpenDepositDoor() error = %v", err)
	}
	if deposited.Status != biz.StatusInStorage || deposited.ExpiresAt == nil {
		t.Fatalf("after deposit status=%s expiresAt=%v", deposited.Status, deposited.ExpiresAt)
	}
	if status, _ := cellState(t, db, order.CellID); status != "OCCUPIED" {
		t.Fatalf("cell status = %s, want OCCUPIED", status)
	}

	// Lazy overdue via the use case exercises MarkOverdue end-to-end.
	uc := biz.NewUseCase(repo, stubDoor{}, func() time.Time { return now })
	if _, err := db.ExecContext(ctx, `UPDATE orders SET expires_at = ? WHERE id = ?`, now.Add(-90*time.Minute), order.ID); err != nil {
		t.Fatalf("backdate expires_at: %v", err)
	}
	lazy, err := uc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder() after expiry error = %v", err)
	}
	if lazy.Status != biz.StatusOverdue {
		t.Fatalf("lazy status = %s, want OVERDUE", lazy.Status)
	}
	if lazy.OverdueFeeFen != 600 {
		t.Fatalf("overdue fee = %d, want 600", lazy.OverdueFeeFen)
	}

	// SettleOverdue -> AWAITING_PICKUP and clears the fee.
	settled, err := repo.SettleOverdue(ctx, order.ID)
	if err != nil {
		t.Fatalf("SettleOverdue() error = %v", err)
	}
	if settled.Status != biz.StatusAwaitingPickup || settled.OverdueFeeFen != 0 {
		t.Fatalf("after settle status=%s fee=%d, want AWAITING_PICKUP 0", settled.Status, settled.OverdueFeeFen)
	}

	// OpenPickupDoor -> COMPLETED and releases the cell.
	completed, err := repo.OpenPickupDoor(ctx, order.ID)
	if err != nil {
		t.Fatalf("OpenPickupDoor() error = %v", err)
	}
	if completed.Status != biz.StatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("after pickup status=%s completedAt=%v", completed.Status, completed.CompletedAt)
	}
	if status, oid := cellState(t, db, order.CellID); status != "IDLE" || oid.Valid {
		t.Fatalf("cell state = %s order=%v, want IDLE released", status, oid)
	}

	// CancelOrder from PENDING_PAYMENT releases the cell.
	cancelOrder, err := repo.CreateOrder(ctx, &biz.Order{
		OrderNo: uniqueSuffix("SPL-"), UserID: biz.DefaultUserID, SiteID: siteID,
		Size: biz.SizeSmall, Status: biz.StatusPendingPayment, DurationMinutes: 60,
		RentFeeFen: 300, TotalFen: 300,
	})
	if err != nil {
		t.Fatalf("second CreateOrder() error = %v", err)
	}
	cancelled, err := repo.CancelOrder(ctx, cancelOrder.ID)
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
	if cancelled.Status != biz.StatusCancelled {
		t.Fatalf("cancel status = %s, want CANCELLED", cancelled.Status)
	}
	if status, oid := cellState(t, db, cancelOrder.CellID); status != "IDLE" || oid.Valid {
		t.Fatalf("cancelled cell state = %s order=%v, want IDLE released", status, oid)
	}
	if _, err := repo.CancelOrder(ctx, cancelOrder.ID); !errors.Is(err, biz.ErrOrderStateConflict) {
		t.Fatalf("repeat CancelOrder error = %v, want ErrOrderStateConflict", err)
	}
}

type stubDoor struct{}

func (stubDoor) OpenDepositDoor(context.Context, string) error { return nil }
func (stubDoor) OpenPickupDoor(context.Context, string) error  { return nil }
