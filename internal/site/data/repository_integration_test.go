//go:build integration

package data

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
)

func TestRepositoryRejectsNilDatabase(t *testing.T) {
	_, err := NewRepository(nil, nil, time.Minute, nil)
	if !errors.Is(err, ErrNilDatabase) {
		t.Fatalf("NewRepository(nil) error = %v, want ErrNilDatabase", err)
	}
}

func TestRepositoryAgainstMigratedSeed(t *testing.T) {
	db := openIntegrationDB(t)
	repo, err := NewRepository(db, nil, 5*time.Minute, func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	ctx := context.Background()

	t.Run("enabled cities and exact lookup", func(t *testing.T) {
		const disabledCode = "TEST-DISABLED-CITY"
		if _, err := db.ExecContext(ctx, `DELETE FROM cities WHERE code = ?`, disabledCode); err != nil {
			t.Fatalf("remove previous test city: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO cities (code, name, province, enabled) VALUES (?, 'disabled', 'test', FALSE)`, disabledCode); err != nil {
			t.Fatalf("insert disabled city: %v", err)
		}
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM cities WHERE code = ?`, disabledCode) })

		cities, err := repo.ListEnabledCities(ctx)
		if err != nil {
			t.Fatalf("ListEnabledCities() error = %v", err)
		}
		if cities == nil || len(cities) == 0 {
			t.Fatalf("ListEnabledCities() = %#v, want seeded enabled city", cities)
		}
		for i, city := range cities {
			if !city.Enabled || city.Code == disabledCode {
				t.Fatalf("ListEnabledCities() included disabled city: %+v", city)
			}
			if i > 0 && (cities[i-1].Code > city.Code || cities[i-1].Code == city.Code && cities[i-1].ID > city.ID) {
				t.Fatalf("ListEnabledCities() is not deterministically ordered: %+v", cities)
			}
		}

		city, err := repo.FindCityByCode(ctx, "310100")
		if err != nil || city.ID != 1 || city.Code != "310100" || !city.Enabled {
			t.Fatalf("FindCityByCode() = %+v, %v", city, err)
		}
		_, err = repo.FindCityByCode(ctx, "31010")
		if !errors.Is(err, biz.ErrCityNotFound) {
			t.Fatalf("partial-code lookup error = %v, want ErrCityNotFound", err)
		}
	})

	t.Run("candidate sites filter bounds and omit phone", func(t *testing.T) {
		const suspendedSiteNo = "TEST-REPO-SUSPENDED"
		_, _ = db.ExecContext(ctx, `DELETE FROM sites WHERE site_no = ?`, suspendedSiteNo)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO sites (site_no, city_id, name, address, latitude, longitude, open_time, close_time, contact_phone, service_status)
			VALUES (?, 1, 'suspended', 'test', 31.2305000, 121.4738000, '08:00:00', '22:00:00', 'distinctive-test-phone', 'SUSPENDED')`, suspendedSiteNo); err != nil {
			t.Fatalf("insert suspended site: %v", err)
		}
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM sites WHERE site_no = ?`, suspendedSiteNo) })

		bounds := biz.GeoBounds{MinLatitude: -90, MaxLatitude: 90, MinLongitude: -180, MaxLongitude: 180}
		sites, err := repo.ListCandidateSites(ctx, 1, bounds)
		if err != nil {
			t.Fatalf("ListCandidateSites() error = %v", err)
		}
		if sites == nil || len(sites) != 2 || sites[0].ID != 1 || sites[1].ID != 2 {
			t.Fatalf("ListCandidateSites() = %+v, want seeded sites [1 2]", sites)
		}
		for _, site := range sites {
			if site.Status != biz.SiteActive || site.ContactPhone != "" {
				t.Fatalf("candidate exposed inactive site or contact phone: %+v", site)
			}
		}

		narrow, err := repo.ListCandidateSites(ctx, 1, biz.GeoBounds{
			MinLatitude: 31.23, MaxLatitude: 31.231, MinLongitude: 121.47, MaxLongitude: 121.474,
		})
		if err != nil || len(narrow) != 1 || narrow[0].ID != 1 {
			t.Fatalf("narrow ListCandidateSites() = %+v, %v", narrow, err)
		}
		empty, err := repo.ListCandidateSites(ctx, 999999, bounds)
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty ListCandidateSites() = %#v, %v", empty, err)
		}
	})

	t.Run("get site includes full fields", func(t *testing.T) {
		site, err := repo.GetSite(ctx, 1)
		if err != nil || site.ID != 1 || site.SiteNo == "" || site.ContactPhone == "" || site.OpenTime == "" || site.Status != biz.SiteActive {
			t.Fatalf("GetSite() = %+v, %v", site, err)
		}
		_, err = repo.GetSite(ctx, 999999)
		if !errors.Is(err, biz.ErrSiteNotFound) {
			t.Fatalf("GetSite(not found) error = %v, want ErrSiteNotFound", err)
		}
	})

	t.Run("availability excludes ineligible devices and cells", func(t *testing.T) {
		prepareSeedDeviceFresh(t, db)
		insertExclusionFixtures(t, db)

		got, err := repo.GetAvailabilitySummary(ctx, 1)
		if err != nil {
			t.Fatalf("GetAvailabilitySummary() error = %v", err)
		}
		want := []biz.CellAvailability{
			{Size: biz.CellSizeSmall, AvailableCount: 3},
			{Size: biz.CellSizeMedium, AvailableCount: 1},
			{Size: biz.CellSizeLarge, AvailableCount: 1},
		}
		if len(got) != len(want) {
			t.Fatalf("GetAvailabilitySummary() = %+v, want %+v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("GetAvailabilitySummary() = %+v, want %+v", got, want)
			}
		}
		empty, err := repo.GetAvailabilitySummary(ctx, 999999)
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty availability = %#v, %v", empty, err)
		}
	})

	t.Run("list cells validates filters and hides unavailable idle cells", func(t *testing.T) {
		prepareSeedDeviceFresh(t, db)
		insertExclusionFixtures(t, db)

		idle, err := repo.ListCells(ctx, 1, "", biz.CellStatusIdle)
		if err != nil {
			t.Fatalf("ListCells(IDLE) error = %v", err)
		}
		if len(idle) != 5 {
			t.Fatalf("ListCells(IDLE) = %+v, want four seed plus one eligible fixture", idle)
		}
		medium, err := repo.ListCells(ctx, 1, biz.CellSizeMedium, biz.CellStatusIdle)
		if err != nil || len(medium) != 1 || medium[0].CellNo != "B01" {
			t.Fatalf("ListCells(MEDIUM, IDLE) = %+v, %v", medium, err)
		}
		locked, err := repo.ListCells(ctx, 1, biz.CellSizeLarge, biz.CellStatusLocked)
		if err != nil || len(locked) != 1 || locked[0].CellNo != "T-LOCKED" {
			t.Fatalf("ListCells(LARGE, LOCKED) = %+v, %v", locked, err)
		}
		empty, err := repo.ListCells(ctx, 999999, "", "")
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty ListCells() = %#v, %v", empty, err)
		}
		if _, err := repo.ListCells(ctx, 1, biz.CellSize("SMALL' OR 1=1 --"), ""); !errors.Is(err, ErrInvalidCellSize) {
			t.Fatalf("invalid size error = %v, want ErrInvalidCellSize", err)
		}
		if _, err := repo.ListCells(ctx, 1, "", biz.CellStatus("IDLE' OR 1=1 --")); !errors.Is(err, ErrInvalidCellStatus) {
			t.Fatalf("invalid status error = %v, want ErrInvalidCellStatus", err)
		}
	})
}

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
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("integration database unavailable")
	}
	return db
}

func prepareSeedDeviceFresh(t *testing.T, db *sql.DB) {
	t.Helper()
	var network, operational string
	var heartbeat sql.NullTime
	if err := db.QueryRow(`SELECT network_status, operational_status, last_heartbeat_at FROM locker_devices WHERE id = 1`).Scan(&network, &operational, &heartbeat); err != nil {
		t.Fatalf("read seed device: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE locker_devices SET network_status = ?, operational_status = ?, last_heartbeat_at = ? WHERE id = 1`, network, operational, heartbeat)
	})
	if _, err := db.Exec(`UPDATE locker_devices SET network_status = 'ONLINE', operational_status = 'ACTIVE', last_heartbeat_at = UTC_TIMESTAMP(6) WHERE id = 1`); err != nil {
		t.Fatalf("freshen seed device: %v", err)
	}
}

func insertExclusionFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	const prefix = "TEST-REPO-"
	_, _ = db.Exec(`DELETE c FROM locker_cells c JOIN locker_devices d ON d.id = c.device_id WHERE d.device_no LIKE ?`, prefix+"%")
	_, _ = db.Exec(`DELETE FROM locker_devices WHERE device_no LIKE ?`, prefix+"%")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE c FROM locker_cells c JOIN locker_devices d ON d.id = c.device_id WHERE d.device_no LIKE ?`, prefix+"%")
		_, _ = db.Exec(`DELETE FROM locker_devices WHERE device_no LIKE ?`, prefix+"%")
	})

	type fixture struct {
		deviceNo, network, operational, heartbeat, cellNo, size, status string
	}
	fixtures := []fixture{
		{prefix + "ELIGIBLE", "ONLINE", "ACTIVE", "UTC_TIMESTAMP(6)", "T-ELIGIBLE", "SMALL", "IDLE"},
		{prefix + "STALE", "ONLINE", "ACTIVE", "UTC_TIMESTAMP(6) - INTERVAL 1 DAY", "T-STALE", "MEDIUM", "IDLE"},
		{prefix + "OFFLINE", "OFFLINE", "ACTIVE", "UTC_TIMESTAMP(6)", "T-OFFLINE", "LARGE", "IDLE"},
		{prefix + "MAINTENANCE", "ONLINE", "MAINTENANCE", "UTC_TIMESTAMP(6)", "T-MAINT", "SMALL", "IDLE"},
		{prefix + "LOCKED", "ONLINE", "ACTIVE", "UTC_TIMESTAMP(6)", "T-LOCKED", "LARGE", "LOCKED"},
	}
	for _, fixture := range fixtures {
		query := `INSERT INTO locker_devices (device_no, site_id, network_status, operational_status, last_heartbeat_at) VALUES (?, 1, ?, ?, ` + fixture.heartbeat + `)`
		result, err := db.Exec(query, fixture.deviceNo, fixture.network, fixture.operational)
		if err != nil {
			t.Fatalf("insert test device: %v", err)
		}
		deviceID, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("test device ID: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO locker_cells (device_id, cell_no, size, occupancy_status) VALUES (?, ?, ?, ?)`, deviceID, fixture.cellNo, fixture.size, fixture.status); err != nil {
			t.Fatalf("insert test cell: %v", err)
		}
	}
}
