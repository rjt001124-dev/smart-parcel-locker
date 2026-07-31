package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	lockerbiz "github.com/rjt001124-dev/smart-parcel-locker/internal/locker/biz"
	lockerdata "github.com/rjt001124-dev/smart-parcel-locker/internal/locker/data"
	platformdata "github.com/rjt001124-dev/smart-parcel-locker/internal/platform/data"
)

var version = "dev"

func main() {
	cfg, err := conf.Load(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clients, err := platformdata.Open(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := clients.Close(); err != nil {
			log.Printf("dependency close failed")
		}
	}()
	repo, err := lockerdata.NewRepository(clients.MySQL, cfg.DeviceOfflineThreshold, nil)
	if err != nil {
		log.Fatal(err)
	}
	useCase := lockerbiz.NewUseCase(repo, lockerdata.NewMarker(clients.Redis), nil)
	reclaimer := lockerbiz.NewReclaimer(useCase, nil, func(count int) {
		log.Printf("expired reservations reclaimed: %d", count)
	})
	log.Printf("reservation worker started version=%s", version)
	if err := reclaimer.Run(ctx); err != nil {
		log.Fatal("reservation worker stopped unexpectedly")
	}
}
