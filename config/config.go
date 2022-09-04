package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Load reads path file to the config object.
func Load(filePath string, config interface{}) error {
	configFile, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "fail to open config file")
	}
	defer configFile.Close()

	return yaml.NewDecoder(configFile).Decode(config)
}
