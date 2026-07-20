package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
)

var (
	ErrNilDatabase       = errors.New("site database is required")
	ErrInvalidCellSize   = errors.New("invalid cell size")
	ErrInvalidCellStatus = errors.New("invalid cell status")
	errDataUnavailable   = errors.New("site data unavailable")
)

const defaultDeviceOfflineThreshold = 5 * time.Minute

type Repository struct {
	db               *sql.DB
	cache            *Cache
	offlineThreshold time.Duration
	now              func() time.Time
}

var _ biz.Repository = (*Repository)(nil)

func NewRepository(db *sql.DB, cache *Cache, offlineThreshold time.Duration, clock func() time.Time) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	if offlineThreshold <= 0 {
		offlineThreshold = defaultDeviceOfflineThreshold
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Repository{db: db, cache: cache, offlineThreshold: offlineThreshold, now: clock}, nil
}

func (r *Repository) ListEnabledCities(ctx context.Context) ([]biz.City, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, name, province, enabled
		FROM cities
		WHERE enabled = TRUE
		ORDER BY code, id`)
	if err != nil {
		return nil, errDataUnavailable
	}
	defer rows.Close()

	cities := make([]biz.City, 0)
	for rows.Next() {
		var city biz.City
		if err := rows.Scan(&city.ID, &city.Code, &city.Name, &city.Province, &city.Enabled); err != nil {
			return nil, errDataUnavailable
		}
		cities = append(cities, city)
	}
	if err := rows.Err(); err != nil {
		return nil, errDataUnavailable
	}
	return cities, nil
}

func (r *Repository) FindCityByCode(ctx context.Context, code string) (biz.City, error) {
	var city biz.City
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, province, enabled
		FROM cities
		WHERE code = ?`, code).Scan(&city.ID, &city.Code, &city.Name, &city.Province, &city.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return biz.City{}, biz.ErrCityNotFound
	}
	if err != nil {
		return biz.City{}, errDataUnavailable
	}
	return city, nil
}

func (r *Repository) ListCandidateSites(ctx context.Context, cityID uint64, bounds biz.GeoBounds) ([]biz.Site, error) {
	if sites, found := r.cache.GetCandidates(ctx, cityID, bounds); found {
		return sites, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.site_no, s.city_id, s.name, s.address, s.latitude, s.longitude,
		       s.open_time, s.close_time, s.service_status
		FROM sites s
		JOIN cities c ON c.id = s.city_id
		WHERE c.id = ? AND c.enabled = TRUE AND s.service_status = 'ACTIVE'
		  AND s.latitude BETWEEN ? AND ? AND s.longitude BETWEEN ? AND ?
		ORDER BY s.id`, cityID, bounds.MinLatitude, bounds.MaxLatitude, bounds.MinLongitude, bounds.MaxLongitude)
	if err != nil {
		return nil, errDataUnavailable
	}
	defer rows.Close()

	sites := make([]biz.Site, 0)
	for rows.Next() {
		var site biz.Site
		if err := rows.Scan(
			&site.ID, &site.SiteNo, &site.CityID, &site.Name, &site.Address,
			&site.Latitude, &site.Longitude, &site.OpenTime, &site.CloseTime, &site.Status,
		); err != nil {
			return nil, errDataUnavailable
		}
		sites = append(sites, site)
	}
	if err := rows.Err(); err != nil {
		return nil, errDataUnavailable
	}
	_ = r.cache.SetCandidates(ctx, cityID, bounds, sites)
	return sites, nil
}

func (r *Repository) GetSite(ctx context.Context, siteID uint64) (biz.Site, error) {
	var site biz.Site
	err := r.db.QueryRowContext(ctx, `
		SELECT id, site_no, city_id, name, address, latitude, longitude,
		       open_time, close_time, contact_phone, service_status
		FROM sites
		WHERE id = ?`, siteID).Scan(
		&site.ID, &site.SiteNo, &site.CityID, &site.Name, &site.Address,
		&site.Latitude, &site.Longitude, &site.OpenTime, &site.CloseTime,
		&site.ContactPhone, &site.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return biz.Site{}, biz.ErrSiteNotFound
	}
	if err != nil {
		return biz.Site{}, errDataUnavailable
	}
	return site, nil
}

func (r *Repository) GetAvailabilitySummary(ctx context.Context, siteID uint64) ([]biz.CellAvailability, error) {
	cutoff := r.now().UTC().Add(-r.offlineThreshold)
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.size, COUNT(*)
		FROM sites s
		JOIN locker_devices d ON d.site_id = s.id
		JOIN locker_cells c ON c.device_id = d.id
		WHERE s.id = ?
		  AND d.network_status = 'ONLINE'
		  AND d.operational_status = 'ACTIVE'
		  AND d.last_heartbeat_at >= ?
		  AND c.occupancy_status = 'IDLE'
		  AND c.size IN ('SMALL', 'MEDIUM', 'LARGE')
		GROUP BY c.size
		ORDER BY CASE c.size WHEN 'SMALL' THEN 1 WHEN 'MEDIUM' THEN 2 WHEN 'LARGE' THEN 3 END`, siteID, cutoff)
	if err != nil {
		return nil, errDataUnavailable
	}
	defer rows.Close()

	availability := make([]biz.CellAvailability, 0, 3)
	for rows.Next() {
		var item biz.CellAvailability
		if err := rows.Scan(&item.Size, &item.AvailableCount); err != nil {
			return nil, errDataUnavailable
		}
		availability = append(availability, item)
	}
	if err := rows.Err(); err != nil {
		return nil, errDataUnavailable
	}
	return availability, nil
}

func (r *Repository) ListCells(ctx context.Context, siteID uint64, size biz.CellSize, status biz.CellStatus) ([]biz.CellView, error) {
	if !validCellSize(size) {
		return nil, ErrInvalidCellSize
	}
	if !validCellStatus(status) {
		return nil, ErrInvalidCellStatus
	}

	query := `
		SELECT c.id, c.cell_no, c.size, c.occupancy_status
		FROM sites s
		JOIN locker_devices d ON d.site_id = s.id
		JOIN locker_cells c ON c.device_id = d.id
		WHERE s.id = ?
		  AND d.operational_status = 'ACTIVE'
		  AND (c.occupancy_status <> 'IDLE'
		       OR (d.network_status = 'ONLINE' AND d.last_heartbeat_at >= ?))`
	args := []any{siteID, r.now().UTC().Add(-r.offlineThreshold)}
	if size != "" {
		query += " AND c.size = ?"
		args = append(args, size)
	}
	if status != "" {
		query += " AND c.occupancy_status = ?"
		args = append(args, status)
	}
	query += " ORDER BY d.id, c.cell_no, c.id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errDataUnavailable
	}
	defer rows.Close()

	cells := make([]biz.CellView, 0)
	for rows.Next() {
		var cell biz.CellView
		if err := rows.Scan(&cell.ID, &cell.CellNo, &cell.Size, &cell.Status); err != nil {
			return nil, errDataUnavailable
		}
		cells = append(cells, cell)
	}
	if err := rows.Err(); err != nil {
		return nil, errDataUnavailable
	}
	return cells, nil
}

func validCellSize(size biz.CellSize) bool {
	switch size {
	case "", biz.CellSizeSmall, biz.CellSizeMedium, biz.CellSizeLarge:
		return true
	default:
		return false
	}
}

func validCellStatus(status biz.CellStatus) bool {
	switch status {
	case "", biz.CellStatusIdle, biz.CellStatusLocked, biz.CellStatusOccupied, biz.CellStatusDisabled:
		return true
	default:
		return false
	}
}
