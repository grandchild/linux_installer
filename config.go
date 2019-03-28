package linux_installer

import (
	"log"

	"gopkg.in/yaml.v2"
)

const configFilename = "config.yml"

// Config holds a list of variables to be expanded in message strings, as well other
// settings for the installer. (Currently only the name of the compressed data file)
type Config struct {
	Variables             StringMap `yaml:"variables,omitempty"`
	DefaultInstallDirName string    `yaml:"default_install_dir_name"`
	StartCommand          string    `yaml:"start_command"`
	IconFile              string    `yaml:"icon_file"`
	DataFilename          string    `yaml:"data_filename"`
	GuiCss                string    `yaml:"gui_css,omitempty"`
}

func ConfigNew() (*Config, error) {
	configFile := MustGetResource(configFilename)
	config := &Config{Variables: make(StringMap)}
	err := yaml.Unmarshal([]byte(configFile), config)
	if err != nil {
		log.Printf("Unable to parse config file %s\n", configFilename)
		return config, err
	}
	return config, err
}
