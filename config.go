package linux_installer

import (
	"log"

	"gopkg.in/yaml.v2"
)

const configFilename = "config.yml"

// Config holds a list of variables to be expanded in message strings, as well other
// settings for the installer.
//
// Setting MustAcceptLicenseOnCli to true (the default) enables & requires the -accept
// flag.
//
// DefaultInstallDirName is a string or template for the default application directory,
// into which to install.
//
// NoLauncher is a flag from the command line that suppresses launcher shortcut
// creation.
type Config struct {
	Variables              VariableMap `yaml:"variables,omitempty"`
	MustAcceptLicenseOnCli bool        `yaml:"must_accept_license_on_cli"`
	DefaultInstallDirName  string      `yaml:"default_install_dir_name"`
	GuiCss                 string      `yaml:"gui_css,omitempty"`

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
