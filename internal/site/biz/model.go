package biz

import "errors"

var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidRadius      = errors.New("invalid radius")
	ErrCityNotFound       = errors.New("city not found")
	ErrSiteNotFound       = errors.New("site not found")
	ErrInvalidSiteID      = errors.New("invalid site ID")
)

type City struct {
	ID       uint64
	Code     string
	Name     string
	Province string
	Enabled  bool
}

type SiteStatus string

const (
	SiteActive    SiteStatus = "ACTIVE"
	SiteSuspended SiteStatus = "SUSPENDED"
	SiteClosed    SiteStatus = "CLOSED"
)

type Site struct {
	ID           uint64
	SiteNo       string
	CityID       uint64
	Name         string
	Address      string
	Latitude     float64
	Longitude    float64
	OpenTime     string
	CloseTime    string
	ContactPhone string
	Status       SiteStatus
}

type NearbyQuery struct {
	CityCode  string
	Latitude  float64
	Longitude float64
	RadiusM   int32
}

type NearbySite struct {
	Site      Site
	DistanceM int32
}

type GeoBounds struct {
	MinLatitude  float64
	MaxLatitude  float64
	MinLongitude float64
	MaxLongitude float64
}

type CellSize string

const (
	CellSizeSmall  CellSize = "SMALL"
	CellSizeMedium CellSize = "MEDIUM"
	CellSizeLarge  CellSize = "LARGE"
)

type CellStatus string

const (
	CellStatusIdle     CellStatus = "IDLE"
	CellStatusLocked   CellStatus = "LOCKED"
	CellStatusOccupied CellStatus = "OCCUPIED"
	CellStatusDisabled CellStatus = "DISABLED"
)

type CellAvailability struct {
	Size           CellSize
	AvailableCount int32
}

type CellView struct {
	ID     uint64
	CellNo string
	Size   CellSize
	Status CellStatus
}

type SiteDetail struct {
	Site         Site
	Availability []CellAvailability
}
