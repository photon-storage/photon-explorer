package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"

	"github.com/photon-storage/photon-explorer/cmd/runtime/version"
	"github.com/photon-storage/photon-explorer/config"
	"github.com/photon-storage/photon-explorer/database/mysql"
	"github.com/photon-storage/photon-explorer/indexer"
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
		log.Error("running application failed", "error", err)
	}
}

func exec(ctx *cli.Context) error {
	cfg := &Config{}
	if err := config.Load(ctx.String(configPathFlag.Name), cfg); err != nil {
		log.Fatal("fail on read config", "error", err)
	}

	db, err := mysql.NewMySQLDB(cfg.MySQL)
	if err != nil {
		log.Fatal("initialize mysql db error", "error", err)
	}

	eventProcessor := indexer.NewEventProcessor(ctx.Context, cfg.RefreshInterval, cfg.NodeEndpoint, db)
	go eventProcessor.Run()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for sig := range c {
		log.Info("event processor stop", "stop_time", time.Now(), "signal", sig.String())
		eventProcessor.Stop()
		return nil
	}

	return nil
}

// Config defines the config for indexer service.
type Config struct {
	MySQL           mysql.Config `json:"mysql"`
	RefreshInterval uint64       `json:"refresh_interval"`
	NodeEndpoint    string       `json:"node_endpoint"`
}
