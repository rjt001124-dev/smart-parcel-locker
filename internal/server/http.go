package server

import (
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	platformdata "github.com/rjt001124-dev/smart-parcel-locker/internal/platform/data"
)

func NewHTTPServer(
	cfg conf.Config,
	version string,
	readiness platformdata.Readiness,
	siteService v1.SiteServiceHTTPServer,
	lockerService v1.InternalLockerServiceHTTPServer,
	deviceService v1.InternalDeviceServiceHTTPServer,
	simulatorService v1.InternalSimulatorServiceHTTPServer,
) *khttp.Server {
	srv := khttp.NewServer(khttp.Address(cfg.HTTPAddr))
	RegisterHealth(srv, version)
	if readiness != nil {
		RegisterReadiness(srv, readiness)
	}
	if siteService != nil {
		v1.RegisterSiteServiceHTTPServer(srv, siteService)
	}
	if lockerService != nil {
		v1.RegisterInternalLockerServiceHTTPServer(srv, lockerService)
	}
	if deviceService != nil {
		v1.RegisterInternalDeviceServiceHTTPServer(srv, deviceService)
	}
	if simulatorService != nil {
		v1.RegisterInternalSimulatorServiceHTTPServer(srv, simulatorService)
	}
	underlying := srv.Server.Handler
	srv.Server.Handler = InternalAuth(cfg.InternalAPIToken)(underlying)
	return srv
}
