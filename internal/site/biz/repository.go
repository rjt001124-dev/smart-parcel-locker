package biz

import "context"

type Repository interface {
	ListEnabledCities(context.Context) ([]City, error)
	FindCityByCode(context.Context, string) (City, error)
	ListCandidateSites(context.Context, uint64, GeoBounds) ([]Site, error)
	GetSite(context.Context, uint64) (Site, error)
	GetAvailabilitySummary(context.Context, uint64) ([]CellAvailability, error)
	ListCells(context.Context, uint64, CellSize, CellStatus) ([]CellView, error)
}
