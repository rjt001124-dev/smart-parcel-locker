package main

import (
	"log"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/server"
)

var version = "dev"

func main() {
	cfg := conf.Load(os.Getenv)
	httpServer := server.NewHTTPServer(cfg, version)

	app := kratos.New(
		kratos.Name("smart-parcel-locker-api"),
		kratos.Version(version),
		kratos.Server(httpServer),
	)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
