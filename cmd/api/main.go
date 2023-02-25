package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"
	pc "github.com/photon-storage/go-photon/config/config"

	"github.com/photon-storage/photon-explorer/api/server"
	"github.com/photon-storage/photon-explorer/api/service"
	"github.com/photon-storage/photon-explorer/cmd/runtime/version"
	"github.com/photon-storage/photon-explorer/config"
	"github.com/photon-storage/photon-explorer/database/mysql"
)

var (
	networkFlag = &cli.StringFlag{
		Name:     "network",
		Usage:    "Network type to boot the node as",
		Required: true,
	}

	// configPathFlag specifies the api config file path.
	configPathFlag = &cli.StringFlag{
		Name:     "config-file",
		Usage:    "The filepath to a json file, flag is required",
		Required: true,
	}

	// logLevelFlag defines the log level.
	logLevelFlag = &cli.StringFlag{
		Name:  "log-level",
		Usage: "Logging verbosity (trace, debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}

	// logFormatFlag specifies the log output format.
	logFormatFlag = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Specify log formatting. Supports: text, json, fluentd, journald.",
		Value: "text",
	}

	// logFilenameFlag specifies the log output file name.
	logFilenameFlag = &cli.StringFlag{
		Name:  "log-file",
		Usage: "Specify log file name, relative or absolute",
	}

	// logColor specifies whether to force log color by skipping TTY check.
	logColorFlag = &cli.BoolFlag{
		Name:  "log-color",
		Usage: "Force log color to be enabled, skipping TTY check",
		Value: false,
	}
)

func main() {
	app := cli.App{
		Name:    "photon-explorer",
		Usage:   "this is a chain explorer implementation for photon",
		Action:  exec,
		Version: version.Get(),
		Flags: []cli.Flag{
			networkFlag,
			configPathFlag,
			logLevelFlag,
			logFormatFlag,
			logFilenameFlag,
			logColorFlag,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logLvl, err := log.ParseLevel(ctx.String(logLevelFlag.Name))
		if err != nil {
			return err
		}

		logFmt, err := log.ParseFormat(ctx.String(logFormatFlag.Name))
		if err != nil {
			return err
		}

		if err := log.Init(logLvl, logFmt, false); err != nil {
			return err
		}

		logFilename := ctx.String(logFilenameFlag.Name)
		if logFilename != "" {
			if err := log.ConfigurePersistentLogging(logFilename, false); err != nil {
				log.Error("Failed to configuring logging to disk",
					"error", err)
			}
		}
		if ctx.Bool(logColorFlag.Name) {
			log.ForceColor()
		}

		configType, err := pc.ConfigTypeFromString(ctx.String(networkFlag.Name))
		if err != nil {
			return err
		}

		return pc.Use(configType)
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

	log.Info("Starting explorer api server...")

	server.New(
		cfg.Port,
		service.New(db, cfg.NodeGatewayProvider),
	).Run()
	return nil
}

// Config defines the config for api service.
type Config struct {
	Port                int          `yaml:"port"`
	MySQL               mysql.Config `yaml:"mysql"`
	NodeGatewayProvider string       `yaml:"node_gateway_provider"`
}
