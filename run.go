package linux_installer

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"strings"
)

const (
	// Linux terminal command string to clear the current line and reset the cursor
	clearLineVT100         = "\033[2K\r"
	cliInstallerMaxLineLen = 80
	logFilename            = "installer.log"
)

// Run parses commandline options (if any) and starts one of two installer modes,
// GUI or commandline mode.
//
// Commandline parameters are:
//   -target          // Target directory to install to
//   -show-license    // Print the software license and exit
//   -accept-license  // Accept the license, needed to install the software in commandline mode.
//   -lang            // Choose install language. This also affects the GUI mode.
//
// Giving any commandline parameters except for the last will trigger commandline, or
// "silent" mode. -target and -accept-license are necessary to run commandline install.
// -lang will also set the default GUI language.
func Run() {
	logfile := startLogging(logFilename)
	defer logfile.Close()

	openBoxes()
	config, err := NewConfig()
	if err != nil {
		return
	}
	config.Variables["installerName"] = os.Args[0]
	translator := NewTranslatorVar(config.Variables)
	installerTempPath := filepath.Join(os.TempDir(), "linux_installer")
	defer os.RemoveAll(installerTempPath)

	target := flag.String("target", "", translator.Get("cli_help_target"))
	showLicense := flag.Bool("license", false, translator.Get("cli_help_showlicense"))
	acceptLicense := flag.Bool("accept", false, translator.Get("cli_help_acceptlicense"))
	noLauncher := flag.Bool("no-launcher", false, translator.Get("cli_help_nolauncher"))
	lang := flag.String("lang", "", translator.Get("cli_help_lang")+" "+strings.Join(translator.GetLanguages(), ", "))
	flag.Parse()

	if len(*lang) > 0 {
		err := translator.SetLanguage(*lang)
		if err != nil {
			fmt.Printf("Language '%s' not available", *lang)
		}
	}

	if *showLicense {
		licenseFile, err := GetResource(
			fmt.Sprintf("licenses/license_%s.txt", translator.language),
		)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Print(licenseFile)
		}
		return
	}

	if *noLauncher {
		config.NoLauncher = true
	}

	if len(*target) > 0 {
		if *acceptLicense {
			RunCliInstall(installerTempPath, *target, translator, config)
		} else {
			fmt.Println(translator.Get("err_cli_mustacceptlicense"))
		}
		return
	}

	guiError := RunGuiInstall(installerTempPath, translator, config)
	if guiError != nil {
		RunTuiInstall(installerTempPath, translator)
	}
}

// RunGuiInstall loads the gui.so plugin, and starts the installer GUI.
func RunGuiInstall(
	installerTempPath string,
	translator *Translator,
	config *Config,
) (err error) {
	UnpackResourceDir("gui", filepath.Join(installerTempPath, "gui"))
	guiPlugin, err := plugin.Open(filepath.Join(installerTempPath, "gui", "gui.so"))
	if err != nil {
		err = osShowRawErrorDialog(translator.Get("err_gui_startup_failed_nogtk"))
		if err != nil {
			handleGuiErr(err, translator.Get("err_gui_startup_failed_nogtk"))
		}
		return
	}
	NewGui, err := guiPlugin.Lookup("NewGui")
	if err != nil {
		osShowRawErrorDialog(translator.Get("err_gui_startup_internal_error"))
		handleGuiErr(err, "")
		return
	}
	RunGui, err := guiPlugin.Lookup("RunGui")
	if err != nil {
		osShowRawErrorDialog(translator.Get("err_gui_startup_internal_error"))
		handleGuiErr(err, "")
		return
	}
	installer := NewInstaller(installerTempPath, config)
	err = NewGui.(func(string, *Installer, *Translator, *Config) error)(
		installerTempPath, installer, translator, config,
	)
	if err != nil {
		handleGuiErr(err, translator.Get("err_gui_startup_failed"))
		return
	} else {
		RunGui.(func())()
	}
	return
}

// RunCliInstall runs a "silent" installation, on the command line with no further user
// interaction.
func RunCliInstall(
	installerTempPath, target string, translator *Translator, config *Config,
) {
	installer := NewInstallerTo(target, installerTempPath, config)
	err := installer.CheckSetInstallDir(target)
	if err != nil {
		log.Println(translator.Get(err.Error()), target)
		fmt.Println(translator.Get(err.Error()))
		return
	}
	installer.CreateLauncher = !config.NoLauncher
	cancelChannel := make(chan os.Signal, 1)
	signal.Notify(cancelChannel, os.Interrupt)
	installer.SetProgressFunction(func(status InstallStatus) {
		file := installer.NextFile().Target
		if len(file) > cliInstallerMaxLineLen {
			file = "..." + file[len(file)-(cliInstallerMaxLineLen-3):]
		}
		fmt.Print(clearLineVT100 + file)
	})
	fmt.Println(translator.Get("silent_installing"))
	installer.PreInstall()
	installer.StartInstall()
	go func() {
		for range cancelChannel {
			installer.Rollback()
		}
	}()
	installer.WaitForDone()
	installer.PostInstall(
		translator.Variables,
		translator.GetAllStringsRaw(),
	)
	fmt.Println(clearLineVT100 + installer.SizeString())
	fmt.Println(translator.Get("silent_done"))
}

// RunTuiInstall starts a terminal curses-based UI (currently disabled).
func RunTuiInstall(installerTempPath string, translator *Translator) {
	// 	tui, err := NewTui(installerTempPath, translator)
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		tui.Run()
	// 	}
}

// startLogging sets up the logging
func startLogging(logFilename string) *os.File {
	logfile, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.Ldate | log.Ltime)
	// log.SetOutput(io.MultiWriter(os.Stdout, logfile))
	log.SetOutput(logfile)
	return logfile
}

// handleGuiErr prints and logs GUI startup errors, and prints the commandline usage.
func handleGuiErr(err error, msg string) {
	log.Println("Unable to load GUI:", err)
	if len(msg) > 0 {
		log.Println(msg)
		fmt.Println(msg)
	}
	flag.PrintDefaults()
}
