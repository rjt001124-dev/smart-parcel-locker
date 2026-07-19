package biz

import (
	"context"
	"errors"
	"math"
	"sort"
)

const (
	earthRadiusM  = 6371008.8
	defaultRadius = int32(5000)
	maxRadius     = int32(50000)
)

type UseCase struct {
	repo Repository
}

type nearbyCandidate struct {
	site      Site
	distanceM float64
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) ListCities(ctx context.Context) ([]City, error) {
	return uc.repo.ListEnabledCities(ctx)
}

func (uc *UseCase) ListNearby(ctx context.Context, query NearbyQuery) ([]NearbySite, error) {
	if !validCoordinates(query.Latitude, query.Longitude) {
		return nil, ErrInvalidCoordinates
	}

	radius, err := normalizeRadius(query.RadiusM)
	if err != nil {
		return nil, err
	}

	city, err := uc.repo.FindCityByCode(ctx, query.CityCode)
	if err != nil {
		if errors.Is(err, ErrCityNotFound) {
			return nil, ErrCityNotFound
		}
		return nil, err
	}
	if city.ID == 0 || !city.Enabled {
		return nil, ErrCityNotFound
	}

	bounds := boundingBox(query.Latitude, query.Longitude, float64(radius))
	candidates, err := uc.repo.ListCandidateSites(ctx, city.ID, bounds)
	if err != nil {
		return nil, err
	}

	matched := make([]nearbyCandidate, 0)
	for _, site := range candidates {
		if site.Status != SiteActive || !validCoordinates(site.Latitude, site.Longitude) {
			continue
		}
		distance := haversineDistanceM(query.Latitude, query.Longitude, site.Latitude, site.Longitude)
		if distance > float64(radius) {
			continue
		}
		matched = append(matched, nearbyCandidate{site: site, distanceM: distance})
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].distanceM == matched[j].distanceM {
			return matched[i].site.ID < matched[j].site.ID
		}
		return matched[i].distanceM < matched[j].distanceM
	})

	nearby := make([]NearbySite, len(matched))
	for i, candidate := range matched {
		nearby[i] = NearbySite{Site: candidate.site, DistanceM: int32(math.Round(candidate.distanceM))}
	}
	return nearby, nil
}

func (uc *UseCase) GetSite(ctx context.Context, siteID uint64) (SiteDetail, error) {
	if siteID == 0 {
		return SiteDetail{}, ErrInvalidSiteID
	}

	site, err := uc.repo.GetSite(ctx, siteID)
	if err != nil {
		if errors.Is(err, ErrSiteNotFound) {
			return SiteDetail{}, ErrSiteNotFound
		}
		return SiteDetail{}, err
	}
	if site.ID == 0 {
		return SiteDetail{}, ErrSiteNotFound
	}

	availability, err := uc.repo.GetAvailabilitySummary(ctx, siteID)
	if err != nil {
		return SiteDetail{}, err
	}
	return SiteDetail{Site: site, Availability: availability}, nil
}

func (uc *UseCase) ListCells(ctx context.Context, siteID uint64, size CellSize, status CellStatus) ([]CellView, error) {
	if siteID == 0 {
		return nil, ErrInvalidSiteID
	}
	return uc.repo.ListCells(ctx, siteID, size, status)
}

func normalizeRadius(radius int32) (int32, error) {
	if radius == 0 {
		return defaultRadius, nil
	}
	if radius < 1 || radius > maxRadius {
		return 0, ErrInvalidRadius
	}
	return radius, nil
}

func validCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) &&
		!math.IsInf(latitude, 0) && !math.IsInf(longitude, 0) &&
		latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func boundingBox(latitude, longitude, radiusM float64) GeoBounds {
	angularRadius := radiusM / earthRadiusM
	latitudeRadians := latitude * math.Pi / 180
	longitudeRadians := longitude * math.Pi / 180
	minLatitudeRadians := clamp(latitudeRadians-angularRadius, -math.Pi/2, math.Pi/2)
	maxLatitudeRadians := clamp(latitudeRadians+angularRadius, -math.Pi/2, math.Pi/2)

	minLongitude, maxLongitude := -180.0, 180.0
	crossesPole := latitudeRadians-angularRadius <= -math.Pi/2 || latitudeRadians+angularRadius >= math.Pi/2
	if !crossesPole {
		ratio := clamp(math.Sin(angularRadius)/math.Cos(latitudeRadians), -1, 1)
		longitudeDelta := math.Asin(ratio)
		if longitudeRadians-longitudeDelta >= -math.Pi && longitudeRadians+longitudeDelta <= math.Pi {
			minLongitude = (longitudeRadians - longitudeDelta) * 180 / math.Pi
			maxLongitude = (longitudeRadians + longitudeDelta) * 180 / math.Pi
		}
	}

	return GeoBounds{
		MinLatitude: minLatitudeRadians * 180 / math.Pi, MaxLatitude: maxLatitudeRadians * 180 / math.Pi,
		MinLongitude: minLongitude, MaxLongitude: maxLongitude,
	}
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

// haversineDistanceM returns the great-circle distance in meters. Callers round
// its result with math.Round, so half meters round away from zero.
func haversineDistanceM(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	latitude1Radians := latitude1 * math.Pi / 180
	latitude2Radians := latitude2 * math.Pi / 180
	latitudeDelta := (latitude2 - latitude1) * math.Pi / 180
	longitudeDelta := (longitude2 - longitude1) * math.Pi / 180

	sinLatitude := math.Sin(latitudeDelta / 2)
	sinLongitude := math.Sin(longitudeDelta / 2)
	a := sinLatitude*sinLatitude + math.Cos(latitude1Radians)*math.Cos(latitude2Radians)*sinLongitude*sinLongitude
	a = clamp(a, 0, 1)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
