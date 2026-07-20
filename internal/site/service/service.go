package service

import (
	"context"
	"errors"
	"reflect"
	"strconv"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
)

var ErrNilUseCase = errors.New("site use case is required")

type UseCase interface {
	ListCities(context.Context) ([]biz.City, error)
	ListNearby(context.Context, biz.NearbyQuery) ([]biz.NearbySite, error)
	GetSite(context.Context, uint64) (biz.SiteDetail, error)
	ListCells(context.Context, uint64, biz.CellSize, biz.CellStatus) ([]biz.CellView, error)
}

type Service struct {
	uc UseCase
}

var _ v1.SiteServiceHTTPServer = (*Service)(nil)

func NewService(uc UseCase) (*Service, error) {
	if uc == nil || isNilUseCase(uc) {
		return nil, ErrNilUseCase
	}
	return &Service{uc: uc}, nil
}

func isNilUseCase(uc UseCase) bool {
	value := reflect.ValueOf(uc)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (s *Service) ListCities(ctx context.Context, _ *v1.ListCitiesRequest) (*v1.ListCitiesReply, error) {
	cities, err := s.uc.ListCities(ctx)
	if err != nil {
		return nil, mapUseCaseError(err)
	}
	reply := &v1.ListCitiesReply{Cities: make([]*v1.City, 0, len(cities))}
	for _, city := range cities {
		reply.Cities = append(reply.Cities, &v1.City{
			Id:       strconv.FormatUint(city.ID, 10),
			Code:     city.Code,
			Name:     city.Name,
			Province: city.Province,
		})
	}
	return reply, nil
}

func (s *Service) ListSites(ctx context.Context, req *v1.ListSitesRequest) (*v1.ListSitesReply, error) {
	if req == nil {
		req = &v1.ListSitesRequest{}
	}
	sites, err := s.uc.ListNearby(ctx, biz.NearbyQuery{
		CityCode:  req.CityCode,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		RadiusM:   req.RadiusM,
	})
	if err != nil {
		return nil, mapUseCaseError(err)
	}
	reply := &v1.ListSitesReply{Sites: make([]*v1.SiteSummary, 0, len(sites))}
	for _, nearby := range sites {
		site := nearby.Site
		reply.Sites = append(reply.Sites, &v1.SiteSummary{
			Id:           strconv.FormatUint(site.ID, 10),
			SiteNo:       site.SiteNo,
			Name:         site.Name,
			Address:      site.Address,
			Latitude:     site.Latitude,
			Longitude:    site.Longitude,
			DistanceM:    nearby.DistanceM,
			Availability: make([]*v1.CellAvailability, 0),
		})
	}
	return reply, nil
}

func (s *Service) GetSite(ctx context.Context, req *v1.GetSiteRequest) (*v1.GetSiteReply, error) {
	var rawID string
	if req != nil {
		rawID = req.SiteId
	}
	siteID, err := parseSiteID(rawID)
	if err != nil {
		return nil, err
	}
	detail, err := s.uc.GetSite(ctx, siteID)
	if err != nil {
		return nil, mapUseCaseError(err)
	}
	site := detail.Site
	return &v1.GetSiteReply{
		Id:                strconv.FormatUint(site.ID, 10),
		SiteNo:            site.SiteNo,
		Name:              site.Name,
		Address:           site.Address,
		Latitude:          site.Latitude,
		Longitude:         site.Longitude,
		OpenTime:          site.OpenTime,
		CloseTime:         site.CloseTime,
		OnlineDeviceCount: 0,
		Availability:      mapAvailability(detail.Availability),
	}, nil
}

func (s *Service) ListCells(ctx context.Context, req *v1.ListCellsRequest) (*v1.ListCellsReply, error) {
	var rawID string
	var size v1.CellSize
	var status v1.CellStatus
	if req != nil {
		rawID = req.SiteId
		size = req.Size
		status = req.Status
	}
	siteID, err := parseSiteID(rawID)
	if err != nil {
		return nil, err
	}
	bizSize, err := toBizCellSize(size)
	if err != nil {
		return nil, err
	}
	bizStatus, err := toBizCellStatus(status)
	if err != nil {
		return nil, err
	}
	cells, err := s.uc.ListCells(ctx, siteID, bizSize, bizStatus)
	if err != nil {
		return nil, mapUseCaseError(err)
	}
	reply := &v1.ListCellsReply{Cells: make([]*v1.CellView, 0, len(cells))}
	for _, cell := range cells {
		reply.Cells = append(reply.Cells, &v1.CellView{
			Id:     strconv.FormatUint(cell.ID, 10),
			CellNo: cell.CellNo,
			Size:   fromBizCellSize(cell.Size),
			Status: fromBizCellStatus(cell.Status),
		})
	}
	return reply, nil
}

func parseSiteID(value string) (uint64, error) {
	siteID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || siteID == 0 {
		return 0, kratoserrors.BadRequest("INVALID_SITE_ID", "invalid site ID")
	}
	return siteID, nil
}

func toBizCellSize(size v1.CellSize) (biz.CellSize, error) {
	switch size {
	case v1.CellSize_CELL_SIZE_UNSPECIFIED:
		return "", nil
	case v1.CellSize_CELL_SIZE_SMALL:
		return biz.CellSizeSmall, nil
	case v1.CellSize_CELL_SIZE_MEDIUM:
		return biz.CellSizeMedium, nil
	case v1.CellSize_CELL_SIZE_LARGE:
		return biz.CellSizeLarge, nil
	default:
		return "", kratoserrors.BadRequest("INVALID_CELL_SIZE", "invalid cell size")
	}
}

func toBizCellStatus(status v1.CellStatus) (biz.CellStatus, error) {
	switch status {
	case v1.CellStatus_CELL_STATUS_UNSPECIFIED:
		return "", nil
	case v1.CellStatus_CELL_STATUS_IDLE:
		return biz.CellStatusIdle, nil
	case v1.CellStatus_CELL_STATUS_LOCKED:
		return biz.CellStatusLocked, nil
	case v1.CellStatus_CELL_STATUS_OCCUPIED:
		return biz.CellStatusOccupied, nil
	case v1.CellStatus_CELL_STATUS_DISABLED:
		return biz.CellStatusDisabled, nil
	default:
		return "", kratoserrors.BadRequest("INVALID_CELL_STATUS", "invalid cell status")
	}
}

func fromBizCellSize(size biz.CellSize) v1.CellSize {
	switch size {
	case biz.CellSizeSmall:
		return v1.CellSize_CELL_SIZE_SMALL
	case biz.CellSizeMedium:
		return v1.CellSize_CELL_SIZE_MEDIUM
	case biz.CellSizeLarge:
		return v1.CellSize_CELL_SIZE_LARGE
	default:
		return v1.CellSize_CELL_SIZE_UNSPECIFIED
	}
}

func fromBizCellStatus(status biz.CellStatus) v1.CellStatus {
	switch status {
	case biz.CellStatusIdle:
		return v1.CellStatus_CELL_STATUS_IDLE
	case biz.CellStatusLocked:
		return v1.CellStatus_CELL_STATUS_LOCKED
	case biz.CellStatusOccupied:
		return v1.CellStatus_CELL_STATUS_OCCUPIED
	case biz.CellStatusDisabled:
		return v1.CellStatus_CELL_STATUS_DISABLED
	default:
		return v1.CellStatus_CELL_STATUS_UNSPECIFIED
	}
}

func mapAvailability(items []biz.CellAvailability) []*v1.CellAvailability {
	availability := make([]*v1.CellAvailability, 0, len(items))
	for _, item := range items {
		availability = append(availability, &v1.CellAvailability{
			Size:           fromBizCellSize(item.Size),
			AvailableCount: item.AvailableCount,
		})
	}
	return availability
}

func mapUseCaseError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return kratoserrors.ClientClosed("REQUEST_CANCELED", "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return kratoserrors.GatewayTimeout("REQUEST_DEADLINE_EXCEEDED", "request deadline exceeded")
	case errors.Is(err, biz.ErrInvalidCoordinates):
		return kratoserrors.BadRequest("INVALID_COORDINATES", "invalid coordinates")
	case errors.Is(err, biz.ErrInvalidRadius):
		return kratoserrors.BadRequest("INVALID_RADIUS", "invalid radius")
	case errors.Is(err, biz.ErrInvalidSiteID):
		return kratoserrors.BadRequest("INVALID_SITE_ID", "invalid site ID")
	case errors.Is(err, biz.ErrCityNotFound):
		return kratoserrors.NotFound("CITY_NOT_FOUND", "city not found")
	case errors.Is(err, biz.ErrSiteNotFound):
		return kratoserrors.NotFound("SITE_NOT_FOUND", "site not found")
	default:
		return kratoserrors.ServiceUnavailable("SITE_DATA_UNAVAILABLE", "site data unavailable")
	}
}
