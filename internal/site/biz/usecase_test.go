package biz

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

type fakeRepository struct {
	cities             []City
	city               City
	candidates         []Site
	site               Site
	availability       []CellAvailability
	cells              []CellView
	listCitiesErr      error
	findCityErr        error
	listCandidatesErr  error
	getSiteErr         error
	availabilityErr    error
	listCellsErr       error
	findCityCalls      int
	listCandidateCalls int
	getSiteCalls       int
	availabilityCalls  int
	listCellsCalls     int
	lastCityCode       string
	lastCityID         uint64
	lastBounds         GeoBounds
	lastSiteID         uint64
	lastCellSiteID     uint64
	lastCellSize       CellSize
	lastCellStatus     CellStatus
	honorBounds        bool
}

func (f *fakeRepository) ListEnabledCities(context.Context) ([]City, error) {
	return f.cities, f.listCitiesErr
}

func (f *fakeRepository) FindCityByCode(_ context.Context, code string) (City, error) {
	f.findCityCalls++
	f.lastCityCode = code
	return f.city, f.findCityErr
}

func (f *fakeRepository) ListCandidateSites(_ context.Context, cityID uint64, bounds GeoBounds) ([]Site, error) {
	f.listCandidateCalls++
	f.lastCityID = cityID
	f.lastBounds = bounds
	if !f.honorBounds || f.listCandidatesErr != nil {
		return f.candidates, f.listCandidatesErr
	}
	candidates := make([]Site, 0, len(f.candidates))
	for _, site := range f.candidates {
		if site.Latitude >= bounds.MinLatitude && site.Latitude <= bounds.MaxLatitude &&
			site.Longitude >= bounds.MinLongitude && site.Longitude <= bounds.MaxLongitude {
			candidates = append(candidates, site)
		}
	}
	return candidates, nil
}

func (f *fakeRepository) GetSite(_ context.Context, siteID uint64) (Site, error) {
	f.getSiteCalls++
	f.lastSiteID = siteID
	return f.site, f.getSiteErr
}

func (f *fakeRepository) GetAvailabilitySummary(context.Context, uint64) ([]CellAvailability, error) {
	f.availabilityCalls++
	return f.availability, f.availabilityErr
}

func (f *fakeRepository) ListCells(_ context.Context, siteID uint64, size CellSize, status CellStatus) ([]CellView, error) {
	f.listCellsCalls++
	f.lastCellSiteID = siteID
	f.lastCellSize = size
	f.lastCellStatus = status
	return f.cells, f.listCellsErr
}

func TestListNearbySitesSortsByDistance(t *testing.T) {
	repo := nearbyRepo([]Site{
		{ID: 2, Name: "far", Latitude: 31.2400, Longitude: 121.4700, Status: SiteActive},
		{ID: 1, Name: "near", Latitude: 31.2305, Longitude: 121.4700, Status: SiteActive},
	})

	got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
		CityCode: "SHA", Latitude: 31.2300, Longitude: 121.4700, RadiusM: 5000,
	})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if len(got) != 2 || got[0].Site.ID != 1 || got[1].Site.ID != 2 {
		t.Fatalf("ListNearby() order = %+v, want site IDs [1 2]", got)
	}
	if got[0].DistanceM != int32(math.Round(haversineDistanceM(31.2300, 121.4700, 31.2305, 121.4700))) {
		t.Fatalf("DistanceM = %d, want nearest-meter rounding", got[0].DistanceM)
	}
}

func TestListNearbyRadiusValidationAndDefault(t *testing.T) {
	tests := []struct {
		name       string
		radius     int32
		wantErr    error
		wantCalls  int
		candidateM float64
		wantCount  int
	}{
		{name: "zero uses five kilometer default", radius: 0, candidateM: 3000, wantCalls: 1, wantCount: 1},
		{name: "maximum accepted", radius: 50000, candidateM: 49000, wantCalls: 1, wantCount: 1},
		{name: "negative rejected", radius: -1, wantErr: ErrInvalidRadius},
		{name: "above maximum rejected", radius: 50001, wantErr: ErrInvalidRadius},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			longitude := tt.candidateM / 111195.0802335329
			repo := nearbyRepo([]Site{{ID: 1, Latitude: 0, Longitude: longitude, Status: SiteActive}})
			got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
				CityCode: "CITY", Latitude: 0, Longitude: 0, RadiusM: tt.radius,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListNearby() error = %v, want %v", err, tt.wantErr)
			}
			if repo.listCandidateCalls != tt.wantCalls {
				t.Fatalf("candidate calls = %d, want %d", repo.listCandidateCalls, tt.wantCalls)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("ListNearby() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestListNearbyRejectsInvalidCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{name: "latitude below range", latitude: -90.1},
		{name: "latitude above range", latitude: 90.1},
		{name: "longitude below range", longitude: -180.1},
		{name: "longitude above range", longitude: 180.1},
		{name: "latitude NaN", latitude: math.NaN()},
		{name: "longitude positive infinity", longitude: math.Inf(1)},
		{name: "latitude negative infinity", latitude: math.Inf(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := nearbyRepo(nil)
			_, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
				CityCode: "CITY", Latitude: tt.latitude, Longitude: tt.longitude, RadiusM: 1000,
			})
			if !errors.Is(err, ErrInvalidCoordinates) {
				t.Fatalf("ListNearby() error = %v, want ErrInvalidCoordinates", err)
			}
			if repo.findCityCalls != 0 {
				t.Fatalf("FindCityByCode calls = %d, want 0", repo.findCityCalls)
			}
		})
	}
}

func TestListNearbyDisabledCityReturnsNotFoundWithoutCandidateQuery(t *testing.T) {
	repo := nearbyRepo(nil)
	repo.city.Enabled = false

	_, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if !errors.Is(err, ErrCityNotFound) {
		t.Fatalf("ListNearby() error = %v, want ErrCityNotFound", err)
	}
	if repo.listCandidateCalls != 0 {
		t.Fatalf("candidate calls = %d, want 0", repo.listCandidateCalls)
	}
}

func TestListNearbyFiltersInactiveAndOutOfRadiusSites(t *testing.T) {
	repo := nearbyRepo([]Site{
		{ID: 1, Latitude: 0, Longitude: 0.001, Status: SiteActive},
		{ID: 2, Latitude: 0, Longitude: 0.001, Status: SiteSuspended},
		{ID: 3, Latitude: 0, Longitude: 0.001, Status: SiteClosed},
		{ID: 4, Latitude: 0, Longitude: 0.009, Status: SiteActive},
	})

	got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if len(got) != 1 || got[0].Site.ID != 1 {
		t.Fatalf("ListNearby() = %+v, want only active in-radius site 1", got)
	}
}

func TestListNearbyEqualRoundedDistanceUsesSiteIDTieBreak(t *testing.T) {
	repo := nearbyRepo([]Site{
		{ID: 9, Latitude: 0, Longitude: -0.001, Status: SiteActive},
		{ID: 3, Latitude: 0, Longitude: 0.001, Status: SiteActive},
	})

	got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if got[0].DistanceM != got[1].DistanceM || got[0].Site.ID != 3 || got[1].Site.ID != 9 {
		t.Fatalf("ListNearby() = %+v, want equal distances ordered by IDs [3 9]", got)
	}
}

func TestListNearbySortsByRawDistanceBeforeRounding(t *testing.T) {
	longitudeAtDistance := func(distanceM float64) float64 {
		return distanceM / earthRadiusM * 180 / math.Pi
	}
	repo := nearbyRepo([]Site{
		{ID: 1, Latitude: 0, Longitude: longitudeAtDistance(100.4), Status: SiteActive},
		{ID: 9, Latitude: 0, Longitude: longitudeAtDistance(100.1), Status: SiteActive},
	})

	got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if got[0].DistanceM != 100 || got[1].DistanceM != 100 {
		t.Fatalf("rounded distances = [%d %d], want [100 100]", got[0].DistanceM, got[1].DistanceM)
	}
	if got[0].Site.ID != 9 || got[1].Site.ID != 1 {
		t.Fatalf("ListNearby() IDs = [%d %d], want nearer site IDs [9 1]", got[0].Site.ID, got[1].Site.ID)
	}
}

func TestListNearbyNoMatchesReturnsNonNilEmptySlice(t *testing.T) {
	got, err := NewUseCase(nearbyRepo(nil)).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("ListNearby() = %#v, want non-nil empty slice", got)
	}
}

func TestListNearbyPropagatesRepositoryErrors(t *testing.T) {
	findErr := errors.New("find city failed")
	repo := nearbyRepo(nil)
	repo.findCityErr = findErr
	_, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if !errors.Is(err, findErr) {
		t.Fatalf("ListNearby() error = %v, want find error", err)
	}

	candidateErr := errors.New("candidate query failed")
	repo = nearbyRepo(nil)
	repo.listCandidatesErr = candidateErr
	_, err = NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{CityCode: "CITY", RadiusM: 1000})
	if !errors.Is(err, candidateErr) {
		t.Fatalf("ListNearby() error = %v, want candidate error", err)
	}
}

func TestListNearbyComputesFiniteClampedBoundsNearPole(t *testing.T) {
	repo := nearbyRepo(nil)
	_, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
		CityCode: "CITY", Latitude: 90, Longitude: 179, RadiusM: 50000,
	})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	bounds := repo.lastBounds
	values := []float64{bounds.MinLatitude, bounds.MaxLatitude, bounds.MinLongitude, bounds.MaxLongitude}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("bounds contain non-finite value: %+v", bounds)
		}
	}
	if bounds.MinLatitude < -90 || bounds.MaxLatitude > 90 || bounds.MinLongitude < -180 || bounds.MaxLongitude > 180 {
		t.Fatalf("bounds are not clamped: %+v", bounds)
	}
}

func TestListNearbyBoundsCoverAntimeridianCrossing(t *testing.T) {
	repo := nearbyRepo(nil)
	_, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
		CityCode: "CITY", Latitude: 0, Longitude: 179.9, RadiusM: 50000,
	})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if repo.lastBounds.MinLongitude != -180 || repo.lastBounds.MaxLongitude != 180 {
		t.Fatalf("longitude bounds = [%f, %f], want full range for antimeridian crossing",
			repo.lastBounds.MinLongitude, repo.lastBounds.MaxLongitude)
	}
}

func TestListNearbyPoleCrossingBoundsUseFullLongitude(t *testing.T) {
	repo := nearbyRepo([]Site{{
		ID: 1, Latitude: 89.8, Longitude: 90, Status: SiteActive,
	}})
	repo.honorBounds = true

	got, err := NewUseCase(repo).ListNearby(context.Background(), NearbyQuery{
		CityCode: "CITY", Latitude: 89.6, Longitude: 0, RadiusM: 50000,
	})
	if err != nil {
		t.Fatalf("ListNearby() error = %v", err)
	}
	if repo.lastBounds.MinLongitude != -180 || repo.lastBounds.MaxLongitude != 180 {
		t.Fatalf("longitude bounds = [%f, %f], want full range for pole crossing",
			repo.lastBounds.MinLongitude, repo.lastBounds.MaxLongitude)
	}
	if len(got) != 1 || got[0].Site.ID != 1 {
		t.Fatalf("ListNearby() = %+v, want in-radius site across pole longitude", got)
	}
}

func TestListCitiesDelegates(t *testing.T) {
	want := []City{{ID: 1, Code: "SHA", Name: "Shanghai", Province: "Shanghai", Enabled: true}}
	repo := &fakeRepository{cities: want}
	got, err := NewUseCase(repo).ListCities(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListCities() = %+v, %v; want %+v, nil", got, err, want)
	}

	repo.listCitiesErr = errors.New("list cities failed")
	_, err = NewUseCase(repo).ListCities(context.Background())
	if !errors.Is(err, repo.listCitiesErr) {
		t.Fatalf("ListCities() error = %v, want repository error", err)
	}
}

func TestGetSiteReturnsAvailability(t *testing.T) {
	repo := &fakeRepository{
		site:         Site{ID: 7, SiteNo: "S-7", Status: SiteActive},
		availability: []CellAvailability{{Size: CellSizeSmall, AvailableCount: 3}},
	}
	got, err := NewUseCase(repo).GetSite(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetSite() error = %v", err)
	}
	if got.Site.ID != 7 || !reflect.DeepEqual(got.Availability, repo.availability) {
		t.Fatalf("GetSite() = %+v, want site and availability", got)
	}
	if repo.lastSiteID != 7 || repo.availabilityCalls != 1 {
		t.Fatalf("repository calls: site ID %d, availability %d", repo.lastSiteID, repo.availabilityCalls)
	}
}

func TestGetSiteValidationAndErrors(t *testing.T) {
	repo := &fakeRepository{}
	_, err := NewUseCase(repo).GetSite(context.Background(), 0)
	if !errors.Is(err, ErrInvalidSiteID) || repo.getSiteCalls != 0 {
		t.Fatalf("GetSite(0) error/calls = %v/%d, want ErrInvalidSiteID/0", err, repo.getSiteCalls)
	}

	repo.getSiteErr = errors.Join(ErrSiteNotFound, errors.New("storage detail"))
	_, err = NewUseCase(repo).GetSite(context.Background(), 8)
	if err != ErrSiteNotFound {
		t.Fatalf("GetSite() error = %v, want stable ErrSiteNotFound", err)
	}

	repo.getSiteErr = nil
	repo.site = Site{ID: 8}
	repo.availabilityErr = errors.New("availability failed")
	_, err = NewUseCase(repo).GetSite(context.Background(), 8)
	if !errors.Is(err, repo.availabilityErr) {
		t.Fatalf("GetSite() error = %v, want availability error", err)
	}
}

func TestListCellsDelegatesAndValidatesSiteID(t *testing.T) {
	want := []CellView{{ID: 11, CellNo: "A01", Size: CellSizeMedium, Status: CellStatusIdle}}
	repo := &fakeRepository{cells: want}
	got, err := NewUseCase(repo).ListCells(context.Background(), 5, CellSizeMedium, CellStatusIdle)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListCells() = %+v, %v; want %+v, nil", got, err, want)
	}
	if repo.lastCellSiteID != 5 || repo.lastCellSize != CellSizeMedium || repo.lastCellStatus != CellStatusIdle {
		t.Fatalf("ListCells() delegated arguments incorrectly")
	}

	_, err = NewUseCase(repo).ListCells(context.Background(), 0, CellSizeSmall, CellStatusIdle)
	if !errors.Is(err, ErrInvalidSiteID) || repo.listCellsCalls != 1 {
		t.Fatalf("ListCells(0) error/calls = %v/%d, want ErrInvalidSiteID/1", err, repo.listCellsCalls)
	}

	repo.listCellsErr = errors.New("list cells failed")
	_, err = NewUseCase(repo).ListCells(context.Background(), 5, CellSizeLarge, CellStatusDisabled)
	if !errors.Is(err, repo.listCellsErr) {
		t.Fatalf("ListCells() error = %v, want repository error", err)
	}
}

func nearbyRepo(candidates []Site) *fakeRepository {
	return &fakeRepository{
		city:       City{ID: 10, Code: "CITY", Enabled: true},
		candidates: candidates,
	}
}
