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

	nearby := make([]NearbySite, 0)
	for _, site := range candidates {
		if site.Status != SiteActive || !validCoordinates(site.Latitude, site.Longitude) {
			continue
		}
		distance := haversineDistanceM(query.Latitude, query.Longitude, site.Latitude, site.Longitude)
		if distance > float64(radius) {
			continue
		}
		nearby = append(nearby, NearbySite{Site: site, DistanceM: int32(math.Round(distance))})
	}

	sort.Slice(nearby, func(i, j int) bool {
		if nearby[i].DistanceM == nearby[j].DistanceM {
			return nearby[i].Site.ID < nearby[j].Site.ID
		}
		return nearby[i].DistanceM < nearby[j].DistanceM
	})
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
	latitudeDelta := angularRadius * 180 / math.Pi
	minLatitude := clamp(latitude-latitudeDelta, -90, 90)
	maxLatitude := clamp(latitude+latitudeDelta, -90, 90)

	minLongitude, maxLongitude := -180.0, 180.0
	cosLatitude := math.Cos(latitude * math.Pi / 180)
	if math.Abs(cosLatitude) > 1e-12 {
		longitudeDelta := latitudeDelta / math.Abs(cosLatitude)
		if longitudeDelta < 180 && longitude-longitudeDelta >= -180 && longitude+longitudeDelta <= 180 {
			minLongitude = longitude - longitudeDelta
			maxLongitude = longitude + longitudeDelta
		}
	}

	return GeoBounds{
		MinLatitude: minLatitude, MaxLatitude: maxLatitude,
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
