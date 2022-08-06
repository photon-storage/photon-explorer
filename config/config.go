package config

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// LoadConfig read path file to the config object
func LoadConfig(config interface{}) error {
	if len(os.Args) <= 1 {
		return errors.New("please setup the config file path")
	}

	configFile, err := os.Open(os.Args[1])
	if err != nil {
		return errors.Wrap(err, "fail to open config file")
	}

	defer configFile.Close()
	return json.NewDecoder(configFile).Decode(config)
}
