package linux_installer

import (
	"log"

	"gopkg.in/yaml.v2"
)

const configFilename = "config.yml"

func ConfigNew() (StringMap, error) {
	configFile := MustGetResource(configFilename)
	config := make(StringMap)
	err := yaml.Unmarshal([]byte(configFile), config)
	if err != nil {
		log.Printf("Unable to parse config file %s\n", configFilename)
		return config, err
	}
	return config, err
}
