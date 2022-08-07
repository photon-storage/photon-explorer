package config

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// LoadConfig reads path file to the config object.
func LoadConfig(filePath string, config interface{}) error {
	configFile, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "fail to open config file")
	}
	defer configFile.Close()

	return json.NewDecoder(configFile).Decode(config)
}
