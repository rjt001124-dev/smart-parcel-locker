package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type stubUseCase struct {
	cities          []biz.City
	citiesErr       error
	nearby          []biz.NearbySite
	nearbyErr       error
	nearbyQuery     biz.NearbyQuery
	nearbyCalls     int
	detail          biz.SiteDetail
	detailErr       error
	getSiteID       uint64
	cells           []biz.CellView
	cellsErr        error
	listCellsSiteID uint64
	listCellsSize   biz.CellSize
	listCellsStatus biz.CellStatus
	listCellsCalls  int
}

func (s *stubUseCase) ListCities(context.Context) ([]biz.City, error) {
	return s.cities, s.citiesErr
}

func (s *stubUseCase) ListNearby(_ context.Context, query biz.NearbyQuery) ([]biz.NearbySite, error) {
	s.nearbyCalls++
	s.nearbyQuery = query
	return s.nearby, s.nearbyErr
}

func (s *stubUseCase) GetSite(_ context.Context, siteID uint64) (biz.SiteDetail, error) {
	s.getSiteID = siteID
	return s.detail, s.detailErr
}

func (s *stubUseCase) ListCells(_ context.Context, siteID uint64, size biz.CellSize, status biz.CellStatus) ([]biz.CellView, error) {
	s.listCellsCalls++
	s.listCellsSiteID = siteID
	s.listCellsSize = size
	s.listCellsStatus = status
	return s.cells, s.cellsErr
}

func TestSiteHTTPHandlerBindsEnumQueries(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantSize   biz.CellSize
		wantStatus biz.CellStatus
	}{
		{name: "symbolic values", query: "size=CELL_SIZE_MEDIUM&status=CELL_STATUS_OCCUPIED", wantSize: biz.CellSizeMedium, wantStatus: biz.CellStatusOccupied},
		{name: "numeric values", query: "size=2&status=3", wantSize: biz.CellSizeMedium, wantStatus: biz.CellStatusOccupied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCase{}
			server := newSiteHTTPServer(t, uc)

			response := serveSiteRequest(server, "/v1/sites/42/cells?"+tt.query)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s, want 200", response.Code, response.Body.String())
			}
			if uc.listCellsSiteID != 42 || uc.listCellsSize != tt.wantSize || uc.listCellsStatus != tt.wantStatus {
				t.Fatalf("ListCells input = (%d, %q, %q), want (42, %q, %q)", uc.listCellsSiteID, uc.listCellsSize, uc.listCellsStatus, tt.wantSize, tt.wantStatus)
			}
		})
	}
}

func TestSiteHTTPHandlerRejectsMalformedQueryValues(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "invalid enum text", path: "/v1/sites/42/cells?size=not-a-cell-size"},
		{name: "malformed numeric value", path: "/v1/sites?radius_m=12x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCase{}
			response := serveSiteRequest(newSiteHTTPServer(t, uc), tt.path)

			assertHTTPError(t, response, http.StatusBadRequest, "CODEC")
			if uc.listCellsCalls != 0 || uc.nearbyCalls != 0 {
				t.Fatalf("use case calls = ListCells %d, ListNearby %d; want no dispatch after binding failure", uc.listCellsCalls, uc.nearbyCalls)
			}
		})
	}
}

func TestSiteHTTPHandlerPathValueOverridesQueryValue(t *testing.T) {
	uc := &stubUseCase{}
	response := serveSiteRequest(newSiteHTTPServer(t, uc), "/v1/sites/42/cells?site_id=99")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s, want 200", response.Code, response.Body.String())
	}
	if uc.listCellsSiteID != 42 {
		t.Fatalf("ListCells site ID = %d, want path value 42", uc.listCellsSiteID)
	}
}

func TestSiteHTTPHandlerReturnsStableSanitizedErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		uc         *stubUseCase
		wantStatus int
		wantReason string
	}{
		{name: "invalid path ID", path: "/v1/sites/not-a-number", uc: &stubUseCase{}, wantStatus: http.StatusBadRequest, wantReason: "INVALID_SITE_ID"},
		{name: "internal use case error", path: "/v1/sites/42/cells", uc: &stubUseCase{cellsErr: errors.New("secret database DSN")}, wantStatus: http.StatusServiceUnavailable, wantReason: "SITE_DATA_UNAVAILABLE"},
		{name: "invalid internal enum", path: "/v1/sites/42/cells", uc: &stubUseCase{cells: []biz.CellView{{ID: 1, CellNo: "A01", Size: biz.CellSize("SECRET_INVALID_SIZE"), Status: biz.CellStatusIdle}}}, wantStatus: http.StatusInternalServerError, wantReason: "INVALID_SITE_DATA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := serveSiteRequest(newSiteHTTPServer(t, tt.uc), tt.path)

			assertHTTPError(t, response, tt.wantStatus, tt.wantReason)
			if strings.Contains(response.Body.String(), "secret database DSN") || strings.Contains(response.Body.String(), "SECRET_INVALID") {
				t.Fatalf("HTTP error leaked internal detail: %s", response.Body.String())
			}
		})
	}
}

func TestNewServiceRejectsNilUseCase(t *testing.T) {
	service, err := NewService(nil)
	if service != nil || !errors.Is(err, ErrNilUseCase) {
		t.Fatalf("NewService(nil) = (%v, %v), want (nil, ErrNilUseCase)", service, err)
	}

	var typedNil *stubUseCase
	service, err = NewService(typedNil)
	if service != nil || !errors.Is(err, ErrNilUseCase) {
		t.Fatalf("NewService(typed nil) = (%v, %v), want (nil, ErrNilUseCase)", service, err)
	}
}

func TestListCitiesMapsOrderedNonNilReply(t *testing.T) {
	uc := &stubUseCase{cities: []biz.City{
		{ID: 9, Code: "SHA", Name: "Shanghai", Province: "Shanghai"},
		{ID: 12, Code: "HGH", Name: "Hangzhou", Province: "Zhejiang"},
	}}
	service := mustService(t, uc)

	reply, err := service.ListCities(context.Background(), &v1.ListCitiesRequest{})
	if err != nil {
		t.Fatalf("ListCities() error = %v", err)
	}
	want := []*v1.City{
		{Id: "9", Code: "SHA", Name: "Shanghai", Province: "Shanghai"},
		{Id: "12", Code: "HGH", Name: "Hangzhou", Province: "Zhejiang"},
	}
	if !reflect.DeepEqual(reply.Cities, want) {
		t.Fatalf("ListCities() cities = %#v, want %#v", reply.Cities, want)
	}

	uc.cities = nil
	reply, err = service.ListCities(context.Background(), &v1.ListCitiesRequest{})
	if err != nil || reply.Cities == nil || len(reply.Cities) != 0 {
		t.Fatalf("ListCities() empty = (%#v, %v), want non-nil empty slice", reply, err)
	}
}

func TestListSitesForwardsQueryAndUsesEmptyAvailabilityWithoutExtraLookup(t *testing.T) {
	uc := &stubUseCase{nearby: []biz.NearbySite{
		{Site: biz.Site{ID: 21, SiteNo: "S-21", Name: "North", Address: "Road 1", Latitude: 31.2, Longitude: 121.5}, DistanceM: 87},
		{Site: biz.Site{ID: 22, SiteNo: "S-22", Name: "South", Address: "Road 2", Latitude: 31.1, Longitude: 121.4}, DistanceM: 231},
	}}
	service := mustService(t, uc)
	req := &v1.ListSitesRequest{CityCode: "SHA", Latitude: 31.23, Longitude: 121.47, RadiusM: 2800}

	reply, err := service.ListSites(context.Background(), req)
	if err != nil {
		t.Fatalf("ListSites() error = %v", err)
	}
	wantQuery := biz.NearbyQuery{CityCode: "SHA", Latitude: 31.23, Longitude: 121.47, RadiusM: 2800}
	if uc.nearbyQuery != wantQuery {
		t.Fatalf("ListNearby query = %#v, want %#v", uc.nearbyQuery, wantQuery)
	}
	if len(reply.Sites) != 2 || reply.Sites[0].Id != "21" || reply.Sites[1].Id != "22" {
		t.Fatalf("ListSites() order/IDs = %#v", reply.Sites)
	}
	first := reply.Sites[0]
	if first.SiteNo != "S-21" || first.Name != "North" || first.Address != "Road 1" || first.Latitude != 31.2 || first.Longitude != 121.5 || first.DistanceM != 87 {
		t.Fatalf("ListSites() first = %#v", first)
	}
	for _, site := range reply.Sites {
		if site.Availability == nil || len(site.Availability) != 0 {
			t.Fatalf("site availability = %#v, want non-nil empty", site.Availability)
		}
	}

	uc.nearby = nil
	reply, err = service.ListSites(context.Background(), req)
	if err != nil || reply.Sites == nil || len(reply.Sites) != 0 {
		t.Fatalf("ListSites() empty = (%#v, %v), want non-nil empty slice", reply, err)
	}
}

func TestGetSiteParsesIDAndMapsPublicFields(t *testing.T) {
	uc := &stubUseCase{detail: biz.SiteDetail{
		Site:         biz.Site{ID: 42, SiteNo: "S-42", Name: "Central", Address: "Main Road", Latitude: 30.1, Longitude: 120.2, OpenTime: "08:00", CloseTime: "22:00", ContactPhone: "secret"},
		Availability: []biz.CellAvailability{{Size: biz.CellSizeSmall, AvailableCount: 3}, {Size: biz.CellSizeLarge, AvailableCount: 1}},
	}}
	service := mustService(t, uc)

	reply, err := service.GetSite(context.Background(), &v1.GetSiteRequest{SiteId: "42"})
	if err != nil {
		t.Fatalf("GetSite() error = %v", err)
	}
	if uc.getSiteID != 42 {
		t.Fatalf("GetSite ID = %d, want 42", uc.getSiteID)
	}
	if reply.Id != "42" || reply.SiteNo != "S-42" || reply.Name != "Central" || reply.Address != "Main Road" || reply.Latitude != 30.1 || reply.Longitude != 120.2 || reply.OpenTime != "08:00" || reply.CloseTime != "22:00" {
		t.Fatalf("GetSite() reply = %#v", reply)
	}
	if reply.OnlineDeviceCount != 0 {
		t.Fatalf("online_device_count = %d, want reserved zero", reply.OnlineDeviceCount)
	}
	wantAvailability := []*v1.CellAvailability{
		{Size: v1.CellSize_CELL_SIZE_SMALL, AvailableCount: 3},
		{Size: v1.CellSize_CELL_SIZE_LARGE, AvailableCount: 1},
	}
	if !reflect.DeepEqual(reply.Availability, wantAvailability) {
		t.Fatalf("availability = %#v, want %#v", reply.Availability, wantAvailability)
	}

	uc.detail.Availability = nil
	reply, err = service.GetSite(context.Background(), &v1.GetSiteRequest{SiteId: "42"})
	if err != nil || reply.Availability == nil || len(reply.Availability) != 0 {
		t.Fatalf("GetSite() nil availability = (%#v, %v), want non-nil empty", reply, err)
	}
}

func TestListCellsForwardsFiltersAndMapsValues(t *testing.T) {
	uc := &stubUseCase{cells: []biz.CellView{
		{ID: 7, CellNo: "A01", Size: biz.CellSizeMedium, Status: biz.CellStatusOccupied},
		{ID: 8, CellNo: "A02", Size: biz.CellSizeMedium, Status: biz.CellStatusLocked},
	}}
	service := mustService(t, uc)

	reply, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "99", Size: v1.CellSize_CELL_SIZE_MEDIUM, Status: v1.CellStatus_CELL_STATUS_OCCUPIED})
	if err != nil {
		t.Fatalf("ListCells() error = %v", err)
	}
	if uc.listCellsSiteID != 99 || uc.listCellsSize != biz.CellSizeMedium || uc.listCellsStatus != biz.CellStatusOccupied {
		t.Fatalf("ListCells forwarding = (%d, %q, %q)", uc.listCellsSiteID, uc.listCellsSize, uc.listCellsStatus)
	}
	want := []*v1.CellView{
		{Id: "7", CellNo: "A01", Size: v1.CellSize_CELL_SIZE_MEDIUM, Status: v1.CellStatus_CELL_STATUS_OCCUPIED},
		{Id: "8", CellNo: "A02", Size: v1.CellSize_CELL_SIZE_MEDIUM, Status: v1.CellStatus_CELL_STATUS_LOCKED},
	}
	if !reflect.DeepEqual(reply.Cells, want) {
		t.Fatalf("ListCells() cells = %#v, want %#v", reply.Cells, want)
	}

	uc.cells = nil
	reply, err = service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "99"})
	if err != nil || uc.listCellsSize != "" || uc.listCellsStatus != "" || reply.Cells == nil || len(reply.Cells) != 0 {
		t.Fatalf("ListCells() unspecified/empty = (%#v, %v), forwarded (%q, %q)", reply, err, uc.listCellsSize, uc.listCellsStatus)
	}
}

func TestResponseMappingRejectsInvalidBusinessEnums(t *testing.T) {
	tests := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "cell size",
			call: func(service *Service) error {
				_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "1"})
				return err
			},
		},
		{
			name: "cell status",
			call: func(service *Service) error {
				_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "1"})
				return err
			},
		},
		{
			name: "availability size",
			call: func(service *Service) error {
				_, err := service.GetSite(context.Background(), &v1.GetSiteRequest{SiteId: "1"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCase{}
			switch tt.name {
			case "cell size":
				uc.cells = []biz.CellView{{ID: 1, CellNo: "A01", Size: biz.CellSize("SECRET_INVALID_SIZE"), Status: biz.CellStatusIdle}}
			case "cell status":
				uc.cells = []biz.CellView{{ID: 1, CellNo: "A01", Size: biz.CellSizeSmall, Status: biz.CellStatus("SECRET_INVALID_STATUS")}}
			case "availability size":
				uc.detail = biz.SiteDetail{Site: biz.Site{ID: 1}, Availability: []biz.CellAvailability{{Size: biz.CellSize("SECRET_INVALID_SIZE"), AvailableCount: 1}}}
			}

			err := tt.call(mustService(t, uc))

			assertPublicError(t, err, http.StatusInternalServerError, "INVALID_SITE_DATA")
			if strings.Contains(err.Error(), "SECRET_INVALID") {
				t.Fatalf("public error leaked invalid internal value: %v", err)
			}
		})
	}
}

func TestRequestValidationErrorsHaveStableCodesAndReasons(t *testing.T) {
	service := mustService(t, &stubUseCase{})
	tests := []struct {
		name   string
		call   func() error
		reason string
	}{
		{name: "empty site ID", call: func() error { _, err := service.GetSite(context.Background(), &v1.GetSiteRequest{}); return err }, reason: "INVALID_SITE_ID"},
		{name: "zero site ID", call: func() error {
			_, err := service.GetSite(context.Background(), &v1.GetSiteRequest{SiteId: "0"})
			return err
		}, reason: "INVALID_SITE_ID"},
		{name: "negative site ID", call: func() error {
			_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "-1"})
			return err
		}, reason: "INVALID_SITE_ID"},
		{name: "non-decimal site ID", call: func() error {
			_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "abc"})
			return err
		}, reason: "INVALID_SITE_ID"},
		{name: "unknown cell size", call: func() error {
			_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "1", Size: v1.CellSize(99)})
			return err
		}, reason: "INVALID_CELL_SIZE"},
		{name: "unknown cell status", call: func() error {
			_, err := service.ListCells(context.Background(), &v1.ListCellsRequest{SiteId: "1", Status: v1.CellStatus(99)})
			return err
		}, reason: "INVALID_CELL_STATUS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPublicError(t, tt.call(), 400, tt.reason)
		})
	}
}

func TestUseCaseErrorsMapToSanitizedPublicErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    int
		reason  string
		message string
	}{
		{name: "coordinates", err: biz.ErrInvalidCoordinates, code: 400, reason: "INVALID_COORDINATES", message: "invalid coordinates"},
		{name: "radius", err: biz.ErrInvalidRadius, code: 400, reason: "INVALID_RADIUS", message: "invalid radius"},
		{name: "site ID", err: biz.ErrInvalidSiteID, code: 400, reason: "INVALID_SITE_ID", message: "invalid site ID"},
		{name: "city missing", err: biz.ErrCityNotFound, code: 404, reason: "CITY_NOT_FOUND", message: "city not found"},
		{name: "site missing", err: biz.ErrSiteNotFound, code: 404, reason: "SITE_NOT_FOUND", message: "site not found"},
		{name: "canceled", err: errors.Join(context.Canceled, errors.New("secret cancellation detail")), code: 499, reason: "REQUEST_CANCELED", message: "request canceled"},
		{name: "deadline", err: errors.Join(context.DeadlineExceeded, errors.New("secret timeout detail")), code: 504, reason: "REQUEST_DEADLINE_EXCEEDED", message: "request deadline exceeded"},
		{name: "other", err: errors.New("secret database DSN"), code: 503, reason: "SITE_DATA_UNAVAILABLE", message: "site data unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapUseCaseError(tt.err)
			assertPublicError(t, err, tt.code, tt.reason)
			if got := kratoserrors.FromError(err).Message; got != tt.message {
				t.Fatalf("message = %q, want %q", got, tt.message)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("public error leaked internal detail: %v", err)
			}
		})
	}
}

func TestSiteProtoContractAndHTTPRoutes(t *testing.T) {
	file := v1.File_locker_v1_site_proto
	service := file.Services().ByName("SiteService")
	if service == nil {
		t.Fatal("SiteService descriptor missing")
	}
	wantRoutes := map[protoreflect.Name]string{
		"ListCities": "/v1/cities",
		"ListSites":  "/v1/sites",
		"GetSite":    "/v1/sites/{site_id}",
		"ListCells":  "/v1/sites/{site_id}/cells",
	}
	if service.Methods().Len() != len(wantRoutes) {
		t.Fatalf("method count = %d, want %d", service.Methods().Len(), len(wantRoutes))
	}
	for methodName, route := range wantRoutes {
		method := service.Methods().ByName(methodName)
		if method == nil {
			t.Fatalf("method %s missing", methodName)
		}
		rule, ok := proto.GetExtension(method.Options(), annotations.E_Http).(*annotations.HttpRule)
		if !ok || rule.GetGet() != route {
			t.Fatalf("%s HTTP route = %#v, want GET %s", methodName, rule, route)
		}
	}
	assertResponseGraphsExcludeSensitiveFields(t, service)

	wantFields := map[protoreflect.Name][]protoreflect.Name{
		"City":             {"id", "code", "name", "province"},
		"SiteSummary":      {"id", "site_no", "name", "address", "latitude", "longitude", "distance_m", "availability"},
		"GetSiteReply":     {"id", "site_no", "name", "address", "latitude", "longitude", "open_time", "close_time", "online_device_count", "availability"},
		"ListCellsRequest": {"site_id", "size", "status"},
		"CellView":         {"id", "cell_no", "size", "status"},
	}
	for messageName, fields := range wantFields {
		message := file.Messages().ByName(messageName)
		if message == nil {
			t.Fatalf("message %s missing", messageName)
		}
		if message.Fields().Len() != len(fields) {
			t.Fatalf("%s field count = %d, want %d", messageName, message.Fields().Len(), len(fields))
		}
		for i, fieldName := range fields {
			field := message.Fields().Get(i)
			if field.Name() != fieldName || field.Number() != protoreflect.FieldNumber(i+1) {
				t.Fatalf("%s field %d = %s/%d, want %s/%d", messageName, i, field.Name(), field.Number(), fieldName, i+1)
			}
		}
	}
}

func assertResponseGraphsExcludeSensitiveFields(t *testing.T, service protoreflect.ServiceDescriptor) {
	t.Helper()
	forbidden := map[protoreflect.Name]struct{}{
		"phone": {}, "contact_phone": {}, "reservation": {}, "device": {}, "device_id": {}, "device_secret": {},
	}
	visited := make(map[protoreflect.FullName]bool)
	var visit func(protoreflect.MessageDescriptor)
	visit = func(message protoreflect.MessageDescriptor) {
		if visited[message.FullName()] {
			return
		}
		visited[message.FullName()] = true
		fields := message.Fields()
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			if _, sensitive := forbidden[field.Name()]; sensitive {
				t.Fatalf("response graph exposes sensitive field %s.%s", message.FullName(), field.Name())
			}
			if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
				visit(field.Message())
			}
		}
	}
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		visit(methods.Get(i).Output())
	}
}

func mustService(t *testing.T, uc UseCase) *Service {
	t.Helper()
	service, err := NewService(uc)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func newSiteHTTPServer(t *testing.T, uc UseCase) *khttp.Server {
	t.Helper()
	service := mustService(t, uc)
	server := khttp.NewServer()
	v1.RegisterSiteServiceHTTPServer(server, service)
	return server
}

func serveSiteRequest(server *khttp.Server, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func assertHTTPError(t *testing.T, response *httptest.ResponseRecorder, code int, reason string) {
	t.Helper()
	body := response.Body.Bytes()
	var public struct {
		Code    int32  `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &public); err != nil {
		t.Fatalf("decode HTTP error %q: %v", body, err)
	}
	if response.Code != code || int(public.Code) != code || public.Reason != reason {
		t.Fatalf("HTTP error = status %d body %+v, want code %d reason %q", response.Code, public, code, reason)
	}
}

func assertPublicError(t *testing.T, err error, code int, reason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %d reason %s", code, reason)
	}
	public := kratoserrors.FromError(err)
	if int(public.Code) != code || public.Reason != reason {
		t.Fatalf("error = code %d reason %q message %q, want code %d reason %q", public.Code, public.Reason, public.Message, code, reason)
	}
}
