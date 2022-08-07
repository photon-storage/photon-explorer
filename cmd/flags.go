package cmd

import "github.com/urfave/cli/v2"

var (
	//ConfigPathFlag specifies the config file path.
	ConfigPathFlag = &cli.StringFlag{
		Name:     "config-file",
		Usage:    "The filepath to a json file, flag is required",
		Required: true,
	}

	// VerbosityFlag defines the log level.
	VerbosityFlag = &cli.StringFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity (trace, debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}

	// LogFormatFlag specifies the log output format.
	LogFormatFlag = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Specify log formatting. Supports: text, json, fluentd, journald.",
		Value: "text",
	}
)
