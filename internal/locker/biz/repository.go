package biz

import (
	"context"
	"time"
)

// Repository is the MySQL-authoritative persistence contract for cell
// reservations. Implementations must enforce device freshness, online status,
// availability, ownership checks, and expired-unbound cleanup atomically.
type Repository interface {
	FindReservation(context.Context, string) (Reservation, error)
	ReserveAvailable(context.Context, ReserveRequest, time.Time) (Reservation, error)
	Release(context.Context, uint64, string) error
	ReclaimExpired(context.Context, time.Time, int) (int, error)
}

// Marker is an optional Redis acceleration layer. It is never authoritative;
// marker failures must not change a successful MySQL reservation outcome.
type Marker interface {
	Mark(context.Context, Reservation) error
	Delete(context.Context, Reservation) error
}
