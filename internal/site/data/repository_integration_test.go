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
	fixedNow := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	repo, err := NewRepository(db, nil, 5*time.Minute, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	ctx := context.Background()

	t.Run("preserves canceled and expired contexts", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := repo.ListEnabledCities(canceledCtx); !errors.Is(err, context.Canceled) {
			t.Fatalf("ListEnabledCities(canceled) error = %v, want context.Canceled", err)
		}

		expiredCtx, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer cancel()
		if _, err := repo.FindCityByCode(expiredCtx, "310100"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("FindCityByCode(expired) error = %v, want context.DeadlineExceeded", err)
		}
	})

	t.Run("nearby index has exact column order", func(t *testing.T) {
		rows, err := db.QueryContext(ctx, `
			SELECT column_name
			FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = 'sites' AND index_name = 'idx_sites_nearby'
			ORDER BY seq_in_index`)
		if err != nil {
			t.Fatalf("query nearby index: %v", err)
		}
		defer rows.Close()
		var got []string
		for rows.Next() {
			var column string
			if err := rows.Scan(&column); err != nil {
				t.Fatalf("scan nearby index column: %v", err)
			}
			got = append(got, column)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate nearby index columns: %v", err)
		}
		want := []string{"city_id", "service_status", "latitude", "longitude", "id"}
		if len(got) != len(want) {
			t.Fatalf("idx_sites_nearby columns = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("idx_sites_nearby columns = %v, want %v", got, want)
			}
		}

		explainRows, err := db.QueryContext(ctx, `
			EXPLAIN SELECT s.id, s.site_no, s.city_id, s.name, s.address, s.latitude, s.longitude,
			       s.open_time, s.close_time, s.service_status
			FROM sites s FORCE INDEX (idx_sites_nearby)
			JOIN cities c ON c.id = s.city_id
			WHERE c.id = ? AND c.enabled = TRUE AND s.service_status = 'ACTIVE'
			  AND s.latitude BETWEEN ? AND ? AND s.longitude BETWEEN ? AND ?
			ORDER BY s.id`, 1, 31.02, 31.03, 121.04, 121.05)
		if err != nil {
			t.Fatalf("explain candidate query: %v", err)
		}
		defer explainRows.Close()
		var siteKey string
		for explainRows.Next() {
			var id int
			var selectType, tableName, accessType string
			var partitions, possibleKeys, key, keyLen, ref, extra sql.NullString
			var estimatedRows int64
			var filtered float64
			if err := explainRows.Scan(&id, &selectType, &tableName, &partitions, &accessType, &possibleKeys, &key, &keyLen, &ref, &estimatedRows, &filtered, &extra); err != nil {
				t.Fatalf("scan candidate explain: %v", err)
			}
			if tableName == "s" && key.Valid {
				siteKey = key.String
			}
		}
		if err := explainRows.Err(); err != nil {
			t.Fatalf("iterate candidate explain: %v", err)
		}
		if siteKey != "idx_sites_nearby" {
			t.Fatalf("candidate query key = %q, want idx_sites_nearby", siteKey)
		}
	})

	t.Run("enabled cities and exact lookup", func(t *testing.T) {
		disabledCode := uniqueFixtureID("TDC-")
		if _, err := db.ExecContext(ctx, `INSERT INTO cities (code, name, province, enabled) VALUES (?, 'disabled', 'test', FALSE)`, disabledCode); err != nil {
			t.Fatalf("insert disabled city: %v", err)
		}
		t.Cleanup(func() { cleanupExec(t, db, "disabled city", `DELETE FROM cities WHERE code = ?`, disabledCode) })

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
		suspendedSiteNo := uniqueFixtureID("TEST-SUSPENDED-")
		if _, err := db.ExecContext(ctx, `
			INSERT INTO sites (site_no, city_id, name, address, latitude, longitude, open_time, close_time, contact_phone, service_status)
			VALUES (?, 1, 'suspended', 'test', 31.2305000, 121.4738000, '08:00:00', '22:00:00', 'distinctive-test-phone', 'SUSPENDED')`, suspendedSiteNo); err != nil {
			t.Fatalf("insert suspended site: %v", err)
		}
		t.Cleanup(func() { cleanupExec(t, db, "suspended site", `DELETE FROM sites WHERE site_no = ?`, suspendedSiteNo) })

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
		prepareSeedDeviceFresh(t, db, fixedNow)
		insertExclusionFixtures(t, db, fixedNow)

		got, err := repo.GetAvailabilitySummary(ctx, 1)
		if err != nil {
			t.Fatalf("GetAvailabilitySummary() error = %v", err)
		}
		want := []biz.CellAvailability{
			{Size: biz.CellSizeSmall, AvailableCount: 3},
			{Size: biz.CellSizeMedium, AvailableCount: 2},
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
		prepareSeedDeviceFresh(t, db, fixedNow)
		fixtures := insertExclusionFixtures(t, db, fixedNow)

		idle, err := repo.ListCells(ctx, 1, "", biz.CellStatusIdle)
		if err != nil {
			t.Fatalf("ListCells(IDLE) error = %v", err)
		}
		if len(idle) != 6 {
			t.Fatalf("ListCells(IDLE) = %+v, want four seed plus eligible and exact-cutoff fixtures", idle)
		}
		medium, err := repo.ListCells(ctx, 1, biz.CellSizeMedium, biz.CellStatusIdle)
		if err != nil || len(medium) != 2 || medium[0].CellNo != "B01" || medium[1].CellNo != fixtures.cutoffCell {
			t.Fatalf("ListCells(MEDIUM, IDLE) = %+v, %v", medium, err)
		}
		locked, err := repo.ListCells(ctx, 1, biz.CellSizeLarge, biz.CellStatusLocked)
		if err != nil || len(locked) != 1 || locked[0].CellNo != fixtures.lockedCell {
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

func prepareSeedDeviceFresh(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()
	var network, operational string
	var heartbeat sql.NullTime
	if err := db.QueryRow(`SELECT network_status, operational_status, last_heartbeat_at FROM locker_devices WHERE id = 1`).Scan(&network, &operational, &heartbeat); err != nil {
		t.Fatalf("read seed device: %v", err)
	}
	t.Cleanup(func() {
		cleanupExec(t, db, "seed device state", `UPDATE locker_devices SET network_status = ?, operational_status = ?, last_heartbeat_at = ? WHERE id = 1`, network, operational, heartbeat)
	})
	if _, err := db.Exec(`UPDATE locker_devices SET network_status = 'ONLINE', operational_status = 'ACTIVE', last_heartbeat_at = ? WHERE id = 1`, now); err != nil {
		t.Fatalf("freshen seed device: %v", err)
	}
}

type exclusionFixtures struct {
	cutoffCell string
	lockedCell string
}

func insertExclusionFixtures(t *testing.T, db *sql.DB, now time.Time) exclusionFixtures {
	t.Helper()
	suffix := uniqueFixtureID("")
	prefix := "TEST-REPO-" + suffix + "-"
	t.Cleanup(func() {
		cleanupExec(t, db, "test cells", `DELETE c FROM locker_cells c JOIN locker_devices d ON d.id = c.device_id WHERE d.device_no LIKE ?`, prefix+"%")
		cleanupExec(t, db, "test devices", `DELETE FROM locker_devices WHERE device_no LIKE ?`, prefix+"%")
	})

	type fixture struct {
		deviceNo, network, operational, cellNo, size, status string
		heartbeat                                            *time.Time
	}
	cutoff := now.Add(-5 * time.Minute)
	stale := cutoff.Add(-time.Microsecond)
	eligibleCell := "T-" + suffix + "-E"
	cutoffCell := "T-" + suffix + "-C"
	lockedCell := "T-" + suffix + "-L"
	fixtures := []fixture{
		{prefix + "ELIGIBLE", "ONLINE", "ACTIVE", eligibleCell, "SMALL", "IDLE", &now},
		{prefix + "CUTOFF", "ONLINE", "ACTIVE", cutoffCell, "MEDIUM", "IDLE", &cutoff},
		{prefix + "STALE", "ONLINE", "ACTIVE", "T-" + suffix + "-S", "MEDIUM", "IDLE", &stale},
		{prefix + "NULL", "ONLINE", "ACTIVE", "T-" + suffix + "-N", "LARGE", "IDLE", nil},
		{prefix + "OFFLINE", "OFFLINE", "ACTIVE", "T-" + suffix + "-O", "LARGE", "IDLE", &now},
		{prefix + "MAINTENANCE", "ONLINE", "MAINTENANCE", "T-" + suffix + "-M", "SMALL", "IDLE", &now},
		{prefix + "LOCKED", "ONLINE", "ACTIVE", lockedCell, "LARGE", "LOCKED", &now},
	}
	for _, fixture := range fixtures {
		result, err := db.Exec(`INSERT INTO locker_devices (device_no, site_id, network_status, operational_status, last_heartbeat_at) VALUES (?, 1, ?, ?, ?)`, fixture.deviceNo, fixture.network, fixture.operational, fixture.heartbeat)
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
	return exclusionFixtures{cutoffCell: cutoffCell, lockedCell: lockedCell}
}

func uniqueFixtureID(prefix string) string {
	return fmt.Sprintf("%s%x", prefix, time.Now().UnixNano())
}

func cleanupExec(t *testing.T, db *sql.DB, label, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Errorf("cleanup %s: %v", label, err)
	}
}
