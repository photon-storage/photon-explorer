package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"

	"github.com/photon-storage/photon-explorer/api/server"
	"github.com/photon-storage/photon-explorer/api/service"
	"github.com/photon-storage/photon-explorer/cmd/runtime/version"
	"github.com/photon-storage/photon-explorer/config"
	"github.com/photon-storage/photon-explorer/database/mysql"
)

func main() {
	app := cli.App{
		Name:    "photon-explorer",
		Usage:   "this is a chain explorer implementation for photon",
		Action:  exec,
		Version: version.Get(),
		Flags: []cli.Flag{
			configPathFlag,
			verbosityFlag,
			logFormatFlag,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logLvl, err := log.ParseLevel(ctx.String(verbosityFlag.Name))
		if err != nil {
			return err
		}

		logFmt, err := log.ParseFormat(ctx.String(logFormatFlag.Name))
		if err != nil {
			return err
		}

		if err := log.Init(logLvl, logFmt); err != nil {
			return err
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error("running api application failed", "error", err)
	}
}

func exec(ctx *cli.Context) error {
	cfg := &Config{}
	if err := config.Load(ctx.String(configPathFlag.Name), cfg); err != nil {
		log.Fatal("reading api config failed", "error", err)
	}

	db, err := mysql.NewMySQLDB(cfg.MySQL)
	if err != nil {
		log.Fatal("initialize mysql db error", "error", err)
	}

	server.New(cfg.Port, service.New(db, cfg.NodeEndpoint)).Run()
	return nil
}

// Config defines the config for api service.
type Config struct {
	Port         int          `json:"port"`
	MySQL        mysql.Config `json:"mysql"`
	NodeEndpoint string       `json:"node_endpoint"`
}
