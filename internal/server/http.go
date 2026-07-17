package server

import (
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
)

func NewHTTPServer(cfg conf.Config, version string) *khttp.Server {
	srv := khttp.NewServer(khttp.Address(cfg.HTTPAddr))
	RegisterHealth(srv, version)
	return srv
}
