package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"
	pc "github.com/photon-storage/go-photon/config/config"

	"github.com/photon-storage/photon-explorer/cmd/runtime/version"
	"github.com/photon-storage/photon-explorer/config"
	"github.com/photon-storage/photon-explorer/database/mysql"
	"github.com/photon-storage/photon-explorer/indexer"
)

var (
	networkFlag = &cli.StringFlag{
		Name:     "network",
		Usage:    "Network type to boot the node as",
		Required: true,
	}

	//configPathFlag specifies the indexer config file path.
	configPathFlag = &cli.StringFlag{
		Name:     "config-file",
		Usage:    "The filepath to a json file, flag is required",
		Required: true,
	}

	// verbosityFlag defines the log level.
	verbosityFlag = &cli.StringFlag{
		Name:  "verbosity",
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
	logColorFlag = &cli.StringFlag{
		Name:  "log-color",
		Usage: "Force log color to be enabled, skipping TTY check",
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
			verbosityFlag,
			logFormatFlag,
			logFilenameFlag,
			logColorFlag,
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

		logFilename := ctx.String(logFilenameFlag.Name)
		if logFilename != "" {
			if err := log.ConfigurePersistentLogging(logFilename, false); err != nil {
				log.Error("Failed to configuring logging to disk",
					"error", err)
			}
		}
		if ctx.IsSet(logColorFlag.Name) {
			log.ForceColor()
		}

		configType, err := pc.ConfigTypeFromString(ctx.String(networkFlag.Name))
		if err != nil {
			return err
		}

		return pc.Use(configType)
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

	eventProcessor := indexer.NewEventProcessor(
		ctx.Context,
		cfg.RefreshInterval,
		cfg.NodeEndpoint,
		db,
	)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")

		go eventProcessor.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic", "times", i-1)
			}
		}
		panic("Panic closing the indexer service")
	}()
	eventProcessor.Run()
	return nil
}

// Config defines the config for indexer service.
type Config struct {
	MySQL           mysql.Config `yaml:"mysql"`
	RefreshInterval uint64       `yaml:"refresh_interval"`
	NodeEndpoint    string       `yaml:"node_endpoint"`
}
