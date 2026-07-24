package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
)

func TestMarkerRoundTripUsesHashedKeyAndSafeJSON(t *testing.T) {
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	m := NewMarker(c)
	r := biz.Reservation{CellID: 7, DeviceID: 2, DeviceNo: "dev", SiteID: 1, CellNo: "A1", Size: biz.SizeMedium, Status: biz.StatusLocked, ReservationKey: "secret-key", ExpiresAt: time.Now().UTC().Add(time.Minute)}
	if err := m.Mark(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	key := markerKey(r.ReservationKey)
	if key == "" || key == "locker:reservation:secret-key" {
		t.Fatalf("bad key %q", key)
	}
	raw, err := s.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if contains(raw, "secret-key") || contains(raw, `"device_no"`) || contains(raw, `"phone"`) {
		t.Fatalf("unsafe marker %s", raw)
	}
	got, err := m.Get(context.Background(), r.ReservationKey)
	if err != nil {
		t.Fatal(err)
	}
	if got.CellID != r.CellID || got.SiteID != r.SiteID {
		t.Fatalf("got %#v", got)
	}
	if ttl := s.TTL(key); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("ttl %v", ttl)
	}
	if err := m.Delete(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if s.Exists(key) {
		t.Fatal("marker remains")
	}
}
func TestMarkerRedisFailureReturned(t *testing.T) {
	m := NewMarker(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	if err := m.Mark(context.Background(), biz.Reservation{ReservationKey: "x", ExpiresAt: time.Now().Add(time.Minute)}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkerExpiresWhenRemainingTTLElapses(t *testing.T) {
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	m := NewMarkerWithClock(c, func() time.Time { return now })
	r := biz.Reservation{CellID: 3, ReservationKey: "expiry-key", ExpiresAt: now.Add(5 * time.Second)}
	if err := m.Mark(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	s.FastForward(6 * time.Second)
	if _, err := m.Get(context.Background(), r.ReservationKey); !errors.Is(err, biz.ErrReservationNotFound) {
		t.Fatalf("Get after expiry error = %v", err)
	}
}
func contains(s, sub string) bool { return len(s) >= len(sub) && (sub == "" || stringContains(s, sub)) }
func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
