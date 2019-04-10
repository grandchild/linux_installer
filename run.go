package linux_installer

import (
	"flag"
	"fmt"
	// "io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

const (
	// Linux terminal command string to clear the current line and reset the cursor
	clearLineVT100         = "\033[2K\r"
	cliInstallerMaxLineLen = 80
)

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
	logfile := startLogging("installer.log")
	defer logfile.Close()

	openBoxes()
	config, err := NewConfig()
	if err != nil {
		return
	}
	translator := NewTranslatorVar(config.Variables)

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

	installerTempPath := filepath.Join(os.TempDir(), "linux_installer")
	defer os.RemoveAll(installerTempPath)
	if len(*target) > 0 {
		if *acceptLicense {
			installer := NewInstallerTo(*target, installerTempPath, config)
			installer.CreateLauncher = !config.NoLauncher
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
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
				for range c {
					installer.Rollback()
				}
			}()
			installer.WaitForDone()
			installer.PostInstall(
				translator.variables,
				translator.GetAllStringsRaw(),
			)
			fmt.Println(clearLineVT100 + installer.SizeString())
			fmt.Println(translator.Get("silent_done"))
		} else {
			fmt.Println(translator.Get("err_cli_mustacceptlicense"))
		}
		return
	}

	var guiError error
	UnpackResourceDir("gui", filepath.Join(installerTempPath, "gui"))
	gui, guiError := NewGui(installerTempPath, translator, config)
	if guiError != nil {
		log.Println("Unable to create window:", guiError)
		fmt.Println(translator.Get("err_gui_startup_failed"))
		flag.PrintDefaults()
	} else {
		gui.Run()
	}
	// if guiError != nil {
	// 	tui, err := NewTui(installerTempPath, translator)
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		tui.Run()
	// 	}
	// }
}
