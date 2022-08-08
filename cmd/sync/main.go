package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"

	"github.com/photon-storage/photon-explorer/cmd"
	"github.com/photon-storage/photon-explorer/cmd/runtime/version"
	"github.com/photon-storage/photon-explorer/config"
	"github.com/photon-storage/photon-explorer/database/mysql"
	"github.com/photon-storage/photon-explorer/sync"
)

func main() {
	app := cli.App{
		Name:    "photon-explorer",
		Usage:   "this is a chain explorer implementation for photon",
		Action:  exec,
		Version: version.Get(),
		Flags: []cli.Flag{
			cmd.ConfigPathFlag,
			cmd.VerbosityFlag,
			cmd.LogFormatFlag,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logLvl, err := log.ParseLevel(ctx.String(cmd.VerbosityFlag.Name))
		if err != nil {
			return err
		}

		logFmt, err := log.ParseFormat(ctx.String(cmd.LogFormatFlag.Name))
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
	cfg := &config.SyncerConfig{}
	if err := config.LoadConfig(ctx.String(cmd.ConfigPathFlag.Name), cfg); err != nil {
		log.Fatal("fail on read config", "error", err)
	}

	db, err := mysql.NewMySQLDB(cfg.MySQL)
	if err != nil {
		log.Fatal("initialize mysql db error", "error", err)
	}

	sync.NewEventProcessor(cfg.SyncSeconds, cfg.NodeURL, db).Run(ctx.Context)
	return nil
}
