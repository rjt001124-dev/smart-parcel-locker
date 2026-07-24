//go:build integration

package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

func TestLockerRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil, time.Minute, nil); !errors.Is(err, ErrNilDatabase) {
		t.Fatalf("NewRepository(nil) error = %v", err)
	}
}

func TestReserveAvailableConcurrentDistinctKeys(t *testing.T) {
	db := openLockerIntegrationDB(t)
	fixture := newLockerFixture(t, db, 1)
	defer fixture.cleanup()
	repo, _ := NewRepository(db, time.Minute, fixture.clock)

	var wg sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: fmt.Sprintf("%s-k-%d", fixture.prefix, i)}, fixture.now.Add(time.Minute))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, biz.ErrCellNotAvailable) {
			t.Fatalf("unexpected reserve error: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful reservations = %d, want 1", success)
	}
}

func TestReserveAvailableSameKeyReplaysReservation(t *testing.T) {
	db := openLockerIntegrationDB(t)
	fixture := newLockerFixture(t, db, 1)
	defer fixture.cleanup()
	repo, _ := NewRepository(db, time.Minute, fixture.clock)
	key := fixture.prefix + "-same"
	wantExpiry := fixture.now.Add(time.Minute)
	first, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: key}, wantExpiry)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: key}, fixture.now.Add(2*time.Minute))
		if err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
		if got.CellID != first.CellID || !got.ExpiresAt.Equal(first.ExpiresAt) {
			t.Fatalf("replay %d = %#v, want %#v", i, got, first)
		}
	}
}

func TestReserveAvailableConcurrentSameKeyReplaysWinner(t *testing.T) {
	db := openLockerIntegrationDB(t)
	fixture := newLockerFixture(t, db, 1)
	defer fixture.cleanup()
	repo, _ := NewRepository(db, time.Minute, fixture.clock)
	key := fixture.prefix + "-concurrent-same"
	wantExpiry := fixture.now.Add(time.Minute)
	var wg sync.WaitGroup
	results := make(chan struct {
		r   biz.Reservation
		err error
	}, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: key}, wantExpiry)
			results <- struct {
				r   biz.Reservation
				err error
			}{r, err}
		}()
	}
	wg.Wait()
	close(results)
	var winner biz.Reservation
	for result := range results {
		if result.err != nil {
			t.Fatalf("same-key request failed: %v", result.err)
		}
		if winner.CellID == 0 {
			winner = result.r
		}
		if result.r.CellID != winner.CellID || !result.r.ExpiresAt.Equal(winner.ExpiresAt) {
			t.Fatalf("replay = %#v, winner = %#v", result.r, winner)
		}
	}
}

func TestReserveRejectsStaleOfflineAndMaintenanceDevices(t *testing.T) {
	db := openLockerIntegrationDB(t)
	fixture := newLockerFixture(t, db, 1)
	defer fixture.cleanup()
	repo, _ := NewRepository(db, time.Minute, fixture.clock)
	for name, update := range map[string]string{"stale": "UPDATE locker_devices SET last_heartbeat_at=? WHERE id=?", "offline": "UPDATE locker_devices SET network_status='OFFLINE' WHERE id=?", "maintenance": "UPDATE locker_devices SET operational_status='MAINTENANCE' WHERE id=?"} {
		fixture.resetCell(t)
		if name == "stale" {
			if _, err := db.Exec(update, fixture.now.Add(-2*time.Minute), fixture.deviceID); err != nil {
				t.Fatal(err)
			}
		} else if _, err := db.Exec(update, fixture.deviceID); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: fixture.prefix + "-" + name}, fixture.now.Add(time.Minute)); !errors.Is(err, biz.ErrCellNotAvailable) {
			t.Fatalf("%s error = %v, want ErrCellNotAvailable", name, err)
		}
		fixture.restoreDevice(t)
	}
}

func TestReleaseOwnershipAndReclaimExpired(t *testing.T) {
	db := openLockerIntegrationDB(t)
	fixture := newLockerFixture(t, db, 1)
	defer fixture.cleanup()
	repo, _ := NewRepository(db, time.Minute, fixture.clock)
	r, err := repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: fixture.prefix + "-release"}, fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Release(context.Background(), r.CellID, "wrong"); !errors.Is(err, biz.ErrReservationKeyMismatch) {
		t.Fatalf("wrong owner error = %v", err)
	}
	if err := repo.Release(context.Background(), r.CellID, r.ReservationKey); err != nil {
		t.Fatal(err)
	}
	r, err = repo.ReserveAvailable(context.Background(), biz.ReserveRequest{SiteID: 1, Size: biz.SizeMedium, ReservationKey: fixture.prefix + "-expired"}, fixture.now.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	count, err := repo.ReclaimExpired(context.Background(), fixture.now, 10)
	if err != nil || count != 1 {
		t.Fatalf("reclaim count=%d err=%v", count, err)
	}
}

type lockerFixture struct {
	db       *sql.DB
	prefix   string
	deviceID uint64
	now      time.Time
}

func (f *lockerFixture) clock() time.Time { return f.now }
func (f *lockerFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE c FROM locker_cells c JOIN locker_devices d ON d.id=c.device_id WHERE d.device_no LIKE ?`, f.prefix+"%")
	_, _ = f.db.Exec(`DELETE FROM locker_devices WHERE device_no LIKE ?`, f.prefix+"%")
}
func (f *lockerFixture) resetCell(t *testing.T) {
	if _, err := f.db.Exec(`UPDATE locker_cells SET occupancy_status='IDLE',reservation_key=NULL,lock_expires_at=NULL,current_order_id=NULL,version=version+1 WHERE device_id=?`, f.deviceID); err != nil {
		t.Fatal(err)
	}
}
func (f *lockerFixture) restoreDevice(t *testing.T) {
	if _, err := f.db.Exec(`UPDATE locker_devices SET network_status='ONLINE',operational_status='ACTIVE',last_heartbeat_at=? WHERE id=?`, f.now, f.deviceID); err != nil {
		t.Fatal(err)
	}
}

func newLockerFixture(t *testing.T, db *sql.DB, cells int) *lockerFixture {
	t.Helper()
	prefix := fmt.Sprintf("it-locker-%d-", time.Now().UnixNano())
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	res, err := db.Exec(`INSERT INTO locker_devices (device_no,site_id,network_status,operational_status,last_heartbeat_at) VALUES (?,1,'ONLINE','ACTIVE',?)`, prefix+"device", now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < cells; i++ {
		if _, err := db.Exec(`INSERT INTO locker_cells (device_id,cell_no,size,occupancy_status) VALUES (?,?,?,'IDLE')`, id, fmt.Sprintf("C%d", i), "MEDIUM"); err != nil {
			t.Fatal(err)
		}
	}
	return &lockerFixture{db: db, prefix: prefix, deviceID: uint64(id), now: now}
}

func openLockerIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}
	if !strings.Contains(dsn, "parseTime=") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
