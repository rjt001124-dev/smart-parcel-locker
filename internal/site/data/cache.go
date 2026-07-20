package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
)

const defaultCandidateCacheTTL = 30 * time.Second

type redisCacheClient interface {
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
	Del(context.Context, ...string) *redis.IntCmd
}

type Cache struct {
	client redisCacheClient
	ttl    time.Duration
}

func NewCache(client redisCacheClient, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = defaultCandidateCacheTTL
	}
	return &Cache{client: client, ttl: ttl}
}

type cachedSite struct {
	ID           uint64         `json:"id"`
	SiteNo       string         `json:"site_no"`
	CityID       uint64         `json:"city_id"`
	Name         string         `json:"name"`
	Address      string         `json:"address"`
	Latitude     float64        `json:"latitude"`
	Longitude    float64        `json:"longitude"`
	OpenTime     string         `json:"open_time"`
	CloseTime    string         `json:"close_time"`
	ContactPhone string         `json:"contact_phone,omitempty"`
	Status       biz.SiteStatus `json:"status"`
}

func candidateCacheKey(cityID uint64, bounds biz.GeoBounds) string {
	stable := fmt.Sprintf("%d|%016x|%016x|%016x|%016x", cityID,
		math.Float64bits(bounds.MinLatitude), math.Float64bits(bounds.MaxLatitude),
		math.Float64bits(bounds.MinLongitude), math.Float64bits(bounds.MaxLongitude))
	digest := sha256.Sum256([]byte(stable))
	return "site:candidates:v1:" + hex.EncodeToString(digest[:])
}

func (c *Cache) GetCandidates(ctx context.Context, cityID uint64, bounds biz.GeoBounds) ([]biz.Site, bool) {
	if c == nil || c.client == nil {
		return nil, false
	}
	key := candidateCacheKey(cityID, bounds)
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var cached []cachedSite
	if err := json.Unmarshal(raw, &cached); err != nil {
		c.evict(ctx, key)
		return nil, false
	}
	if cached == nil {
		c.evict(ctx, key)
		return nil, false
	}
	sites := make([]biz.Site, 0, len(cached))
	for _, site := range cached {
		if !validCachedSite(site) {
			c.evict(ctx, key)
			return nil, false
		}
		sites = append(sites, biz.Site{
			ID: site.ID, SiteNo: site.SiteNo, CityID: site.CityID, Name: site.Name,
			Address: site.Address, Latitude: site.Latitude, Longitude: site.Longitude,
			OpenTime: site.OpenTime, CloseTime: site.CloseTime, Status: site.Status,
		})
	}
	return sites, true
}

func validCachedSite(site cachedSite) bool {
	return site.ID > 0 && site.CityID > 0 && site.SiteNo != "" && site.Name != "" &&
		!math.IsNaN(site.Latitude) && !math.IsInf(site.Latitude, 0) &&
		!math.IsNaN(site.Longitude) && !math.IsInf(site.Longitude, 0) &&
		site.Latitude >= -90 && site.Latitude <= 90 &&
		site.Longitude >= -180 && site.Longitude <= 180 &&
		site.Status == biz.SiteActive && site.ContactPhone == ""
}

func (c *Cache) evict(ctx context.Context, key string) {
	_ = c.client.Del(ctx, key).Err()
}

func (c *Cache) SetCandidates(ctx context.Context, cityID uint64, bounds biz.GeoBounds, sites []biz.Site) error {
	if c == nil || c.client == nil {
		return nil
	}
	cached := make([]cachedSite, 0, len(sites))
	for _, site := range sites {
		cached = append(cached, cachedSite{
			ID: site.ID, SiteNo: site.SiteNo, CityID: site.CityID, Name: site.Name,
			Address: site.Address, Latitude: site.Latitude, Longitude: site.Longitude,
			OpenTime: site.OpenTime, CloseTime: site.CloseTime, Status: site.Status,
		})
	}
	payload, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("encode candidate site cache: %w", err)
	}
	return c.client.Set(ctx, candidateCacheKey(cityID, bounds), payload, c.ttl).Err()
}
