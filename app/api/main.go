package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	devicebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
	devicedata "github.com/rjt001124-dev/smart-parcel-locker/internal/device/data"
	deviceservice "github.com/rjt001124-dev/smart-parcel-locker/internal/device/service"
	lockerbiz "github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
	lockerdata "github.com/rjt001124-dev/smart-parcel-locker/internal/locker/data"
	lockerservice "github.com/rjt001124-dev/smart-parcel-locker/internal/locker/service"
	platformdata "github.com/rjt001124-dev/smart-parcel-locker/internal/platform/data"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/server"
	sitebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/site/biz"
	sitedata "github.com/rjt001124-dev/smart-parcel-locker/internal/site/data"
	siteservice "github.com/rjt001124-dev/smart-parcel-locker/internal/site/service"
)

var version = "dev"

func main() {
	cfg, err := conf.Load(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.AppEnv != "test" && strings.TrimSpace(cfg.InternalAPIToken) == "" {
		log.Fatal("INTERNAL_API_TOKEN is required")
	}
	clients, err := platformdata.Open(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := clients.Close(); closeErr != nil {
			log.Printf("dependency close failed")
		}
	}()

	cache := sitedata.NewCache(clients.Redis, 30*time.Second)
	siteRepo, err := sitedata.NewRepository(clients.MySQL, cache, cfg.DeviceOfflineThreshold, nil)
	if err != nil {
		log.Fatal(err)
	}
	siteService, err := siteservice.NewService(sitebiz.NewUseCase(siteRepo))
	if err != nil {
		log.Fatal(err)
	}

	lockerRepo, err := lockerdata.NewRepository(clients.MySQL, cfg.DeviceOfflineThreshold, nil)
	if err != nil {
		log.Fatal(err)
	}
	lockerUseCase := lockerbiz.NewUseCase(lockerRepo, lockerdata.NewMarker(clients.Redis), nil)
	lockerService := lockerservice.NewService(lockerUseCase)

	deviceRepo, err := devicedata.NewRepository(clients.MySQL)
	if err != nil {
		log.Fatal(err)
	}
	simulator := devicedata.NewSimulator()
	deviceUseCase := devicebiz.NewUseCase(deviceRepo, simulator, cfg.DeviceOfflineThreshold)
	deviceService := deviceservice.NewService(deviceUseCase, func(deviceNo, scenario string) error {
		return simulator.SetScenario(deviceNo, devicedata.Scenario(scenario))
	}, cfg.AppEnv != "production", nil)

	httpServer := server.NewHTTPServer(
		cfg,
		version,
		platformdata.NewReadiness(clients),
		siteService,
		lockerService,
		deviceService,
	)

	app := kratos.New(
		kratos.Name("smart-parcel-locker-api"),
		kratos.Version(version),
		kratos.Server(httpServer),
	)
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}
