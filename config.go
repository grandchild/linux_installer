package linux_installer

import (
	"log"

	"gopkg.in/yaml.v2"
)

const configFilename = "config.yml"

// Config holds a list of variables to be expanded in message strings, as well other
// settings for the installer.
//
// StartCommand is the name of the executable file that starts the installed program.
//
// IconFile is needed for the launcher shortcut creation and should be a filename or
// filepath relative to the install directory.
//
// DataFilename is the name of the compressed zip-file that needs to be extracted before
// installation and contains all data for the program to be installed.
//
// GuiCss is a CSS string configuring the style of the installer GUI.
//
// NoLauncher is a flag from the command line that suppresses launcher shortcut
// creation.
type Config struct {
	Variables             VariableMap `yaml:"variables,omitempty"`
	DefaultInstallDirName string      `yaml:"default_install_dir_name"`
	StartCommand          string      `yaml:"start_command"`
	IconFile              string      `yaml:"icon_file"`
	DataFilename          string      `yaml:"data_filename"`
	GuiCss                string      `yaml:"gui_css,omitempty"`

	// commandline config options
	NoLauncher bool
}

// NewConfig returns a Config object containing the settings from resources/config.yml.
func NewConfig() (*Config, error) {
	configFile := MustGetResource(configFilename)
	config := &Config{Variables: make(VariableMap)}
	err := yaml.Unmarshal([]byte(configFile), config)
	if err != nil {
		log.Printf("Unable to parse config file %s\n", configFilename)
		return config, err
	}
	return config, err
}
