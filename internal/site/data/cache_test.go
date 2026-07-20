package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
)

func TestCandidateCacheKeyIsStableAndSensitiveToInputs(t *testing.T) {
	bounds := biz.GeoBounds{MinLatitude: -1, MaxLatitude: 2, MinLongitude: -180, MaxLongitude: 180}

	first := candidateCacheKey(7, bounds)
	second := candidateCacheKey(7, bounds)
	if first != second {
		t.Fatalf("same inputs produced different keys: %q and %q", first, second)
	}
	if !strings.HasPrefix(first, "site:candidates:v1:") || len(strings.TrimPrefix(first, "site:candidates:v1:")) != 64 {
		t.Fatalf("key is not a SHA-256 candidate key: %q", first)
	}
	if first == candidateCacheKey(8, bounds) {
		t.Fatal("different city IDs produced the same key")
	}
	changedBounds := bounds
	changedBounds.MaxLongitude = 179.5
	if first == candidateCacheKey(7, changedBounds) {
		t.Fatal("different bounds produced the same key")
	}
}

func TestCacheRejectsAndEvictsInvalidCandidatePayloads(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "JSON null", payload: `null`},
		{name: "missing required identity", payload: `[{"id":1,"city_id":9,"name":"site","latitude":31.2,"longitude":121.4,"status":"ACTIVE"}]`},
		{name: "invalid coordinates", payload: `[{"id":1,"site_no":"S-1","city_id":9,"name":"site","latitude":91,"longitude":121.4,"status":"ACTIVE"}]`},
		{name: "unknown status", payload: `[{"id":1,"site_no":"S-1","city_id":9,"name":"site","latitude":31.2,"longitude":121.4,"status":"UNKNOWN"}]`},
		{name: "non-active status", payload: `[{"id":1,"site_no":"S-1","city_id":9,"name":"site","latitude":31.2,"longitude":121.4,"status":"SUSPENDED"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr()})
			t.Cleanup(func() { _ = client.Close() })
			cache := NewCache(client, time.Minute)
			bounds := biz.GeoBounds{MinLatitude: 30, MaxLatitude: 32, MinLongitude: 120, MaxLongitude: 122}
			key := candidateCacheKey(9, bounds)
			if err := client.Set(context.Background(), key, tt.payload, time.Minute).Err(); err != nil {
				t.Fatalf("seed invalid cache: %v", err)
			}

			got, found := cache.GetCandidates(context.Background(), 9, bounds)
			if found || got != nil {
				t.Fatalf("GetCandidates() = %#v, %v; want nil, false", got, found)
			}
			if server.Exists(key) {
				t.Fatal("invalid cache value was not evicted")
			}
		})
	}
}

func TestCacheRoundTripUsesDefaultTTLAndOmitsContactPhone(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewCache(client, 0)
	ctx := context.Background()
	bounds := biz.GeoBounds{MinLatitude: 30, MaxLatitude: 32, MinLongitude: 120, MaxLongitude: 122}
	sites := []biz.Site{{
		ID: 1, SiteNo: "S-1", CityID: 9, Name: "site", Address: "address",
		Latitude: 31.2, Longitude: 121.4, OpenTime: "08:00:00", CloseTime: "22:00:00",
		ContactPhone: "DISTINCTIVE-PHONE-MARKER", Status: biz.SiteActive,
	}}

	if err := cache.SetCandidates(ctx, 9, bounds, sites); err != nil {
		t.Fatalf("SetCandidates() error = %v", err)
	}
	got, found := cache.GetCandidates(ctx, 9, bounds)
	if !found {
		t.Fatal("GetCandidates() found = false, want true")
	}
	if len(got) != 1 || got[0].ID != sites[0].ID || got[0].ContactPhone != "" || got[0].Status != biz.SiteActive {
		t.Fatalf("GetCandidates() = %#v", got)
	}

	key := candidateCacheKey(9, bounds)
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("read cached value: %v", err)
	}
	if strings.Contains(raw, "DISTINCTIVE-PHONE-MARKER") || strings.Contains(raw, "ContactPhone") {
		t.Fatalf("cached value contains contact phone data: %q", raw)
	}
	if ttl := server.TTL(key); ttl != 30*time.Second {
		t.Fatalf("TTL = %v, want %v", ttl, 30*time.Second)
	}
}

func TestCacheMissesAreNonfatal(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		server := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		got, found := NewCache(client, time.Minute).GetCandidates(context.Background(), 1, biz.GeoBounds{})
		if found || got != nil {
			t.Fatalf("GetCandidates() = %#v, %v; want nil, false", got, found)
		}
	})

	t.Run("redis unavailable", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{
			Addr:        "127.0.0.1:1",
			DialTimeout: 10 * time.Millisecond,
			ReadTimeout: 10 * time.Millisecond,
		})
		t.Cleanup(func() { _ = client.Close() })
		got, found := NewCache(client, time.Minute).GetCandidates(context.Background(), 1, biz.GeoBounds{})
		if found || got != nil {
			t.Fatalf("GetCandidates() = %#v, %v; want nil, false", got, found)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		server := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		cache := NewCache(client, time.Minute)
		key := candidateCacheKey(1, biz.GeoBounds{})
		if err := client.Set(context.Background(), key, "not-json", time.Minute).Err(); err != nil {
			t.Fatalf("seed malformed cache: %v", err)
		}

		got, found := cache.GetCandidates(context.Background(), 1, biz.GeoBounds{})
		if found || got != nil {
			t.Fatalf("GetCandidates() = %#v, %v; want nil, false", got, found)
		}
		if server.Exists(key) {
			t.Fatal("malformed cache value was not evicted")
		}
	})
}
