package biz

import (
	"errors"
	"time"
)

const (
	// DefaultReservationTTL is used when a reserve request omits a TTL.
	DefaultReservationTTL = 5 * time.Minute
	MinReservationTTL     = 30 * time.Second
	MaxReservationTTL     = 15 * time.Minute
)

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrInvalidReservationTTL  = errors.New("invalid reservation TTL")
	ErrReservationNotFound    = errors.New("reservation not found")
	ErrDeviceOffline          = errors.New("device offline")
	ErrCellNotAvailable       = errors.New("cell not available")
	ErrReservationKeyMismatch = errors.New("reservation key mismatch")
	ErrInvalidCellID          = errors.New("invalid cell ID")
)

// ErrNoAvailableCell is kept as a descriptive alias for repositories that
// report the stable cell-not-available domain condition by that name.
var ErrNoAvailableCell = ErrCellNotAvailable

// ErrCellReservationConflict is an alias for a failed ownership check.
var ErrCellReservationConflict = ErrReservationKeyMismatch

type CellSize string

const (
	SizeUnspecified CellSize = ""
	SizeSmall       CellSize = "SMALL"
	SizeMedium      CellSize = "MEDIUM"
	SizeLarge       CellSize = "LARGE"

	// Compatibility aliases make the domain names explicit at call sites that
	// prefer the CellSize prefix while retaining the concise API names.
	CellSizeUnspecified = SizeUnspecified
	CellSizeSmall       = SizeSmall
	CellSizeMedium      = SizeMedium
	CellSizeLarge       = SizeLarge
)

type CellStatus string

const (
	StatusUnspecified CellStatus = ""
	StatusIdle        CellStatus = "IDLE"
	StatusLocked      CellStatus = "LOCKED"
	StatusOccupied    CellStatus = "OCCUPIED"
	StatusDisabled    CellStatus = "DISABLED"

	CellStatusUnspecified = StatusUnspecified
	CellStatusIdle        = StatusIdle
	CellStatusLocked      = StatusLocked
	CellStatusOccupied    = StatusOccupied
	CellStatusDisabled    = StatusDisabled
)

// Reservation is the MySQL-authoritative result of a cell reservation.
// Device and cell metadata are optional and are populated by repositories that
// need to map the result directly to an internal API response.
type Reservation struct {
	CellID         uint64
	DeviceID       uint64
	DeviceNo       string
	SiteID         uint64
	CellNo         string
	Size           CellSize
	Status         CellStatus
	ReservationKey string
	ExpiresAt      time.Time
}

type ReserveRequest struct {
	SiteID         uint64
	Size           CellSize
	ReservationKey string
	TTL            time.Duration
}
