package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

type Marker struct {
	client redis.UniversalClient
	now    func() time.Time
}

var _ biz.Marker = (*Marker)(nil)

type safeMarker struct {
	CellID    uint64         `json:"cell_id"`
	DeviceID  uint64         `json:"device_id,omitempty"`
	SiteID    uint64         `json:"site_id,omitempty"`
	CellNo    string         `json:"cell_no,omitempty"`
	Size      biz.CellSize   `json:"size,omitempty"`
	Status    biz.CellStatus `json:"status,omitempty"`
	ExpiresAt time.Time      `json:"expires_at"`
}

func NewMarker(client redis.UniversalClient) *Marker {
	return NewMarkerWithClock(client, nil)
}

// NewMarkerWithClock is useful for deterministic expiry tests while retaining
// UTC as the production default.
func NewMarkerWithClock(client redis.UniversalClient, clock func() time.Time) *Marker {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Marker{client: client, now: clock}
}
func markerKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return "locker:reservation:" + hex.EncodeToString(h[:])
}
func (m *Marker) Mark(ctx context.Context, r biz.Reservation) error {
	if m == nil || m.client == nil {
		return errors.New("redis unavailable")
	}
	ttl := r.ExpiresAt.UTC().Sub(m.now().UTC())
	if ttl <= 0 {
		return nil
	}
	b, err := json.Marshal(safeMarker{
		CellID:    r.CellID,
		DeviceID:  r.DeviceID,
		SiteID:    r.SiteID,
		CellNo:    r.CellNo,
		Size:      r.Size,
		Status:    r.Status,
		ExpiresAt: r.ExpiresAt.UTC(),
	})
	if err != nil {
		return err
	}
	return m.client.Set(ctx, markerKey(r.ReservationKey), b, ttl).Err()
}
func (m *Marker) Delete(ctx context.Context, r biz.Reservation) error {
	if m == nil || m.client == nil {
		return errors.New("redis unavailable")
	}
	return m.client.Del(ctx, markerKey(r.ReservationKey)).Err()
}
func (m *Marker) Get(ctx context.Context, key string) (biz.Reservation, error) {
	if m == nil || m.client == nil {
		return biz.Reservation{}, errors.New("redis unavailable")
	}
	b, err := m.client.Get(ctx, markerKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return biz.Reservation{}, biz.ErrReservationNotFound
	}
	if err != nil {
		return biz.Reservation{}, err
	}
	var s safeMarker
	if err := json.Unmarshal(b, &s); err != nil {
		return biz.Reservation{}, err
	}
	return biz.Reservation{CellID: s.CellID, DeviceID: s.DeviceID, SiteID: s.SiteID, CellNo: s.CellNo, Size: s.Size, Status: s.Status, ReservationKey: key, ExpiresAt: s.ExpiresAt}, nil
}
