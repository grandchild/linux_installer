package linux_installer

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

const (
	// Linux terminal command string to clear the current line and reset the cursor
	clearLineVT100 = "\033[2K\r"
)

func startLogging(logFilename string) *os.File {
	logfile, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(io.MultiWriter(os.Stdout, logfile))
	return logfile
}

func Run() {
	logfile := startLogging("installer.log")
	defer logfile.Close()

	openBoxes()
	config, err := ConfigNew()
	if err != nil {
		return
	}
	translator := NewTranslatorVar(config)

	// cli := flag.Bool("cli", false, translator.Get("cli_help_nogui"))
	target := flag.String("target", "", translator.Get("cli_help_target"))
	showLicense := flag.Bool("show-license", false, translator.Get("cli_help_showlicense"))
	acceptLicense := flag.Bool("accept-license", false, translator.Get("cli_help_acceptlicense"))
	lang := flag.String("lang", "", translator.Get("cli_help_lang")+" "+strings.Join(translator.GetLanguageOptionKeys(), ", "))
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

	installerTempPath := filepath.Join(os.TempDir(), "linux_installer")
	if len(*target) > 0 {
		if *acceptLicense {
			installer := InstallerToNew(*target, installerTempPath)
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			installer.SetProgressFunction(func(status InstallStatus) {
				file := installer.NextFile().Target
				maxLen := 70
				if len(file) > maxLen {
					file = "..." + file[len(file)-(maxLen-3):]
				}
				fmt.Print(clearLineVT100 + file)
			})
			fmt.Println(translator.Get("silent_installing"))
			installer.StartInstall()
			go func() {
				for range c {
					installer.Rollback()
				}
			}()
			installer.WaitForDone()
			fmt.Println(clearLineVT100 + installer.SizeString())
			fmt.Println(translator.Get("silent_done"))
		} else {
			fmt.Println(translator.Get("err_cli_mustacceptlicense"))
		}
		return
	}

	var guiError error
	defer os.RemoveAll(installerTempPath)
	// if !*cli {
	UnpackResourceDir("gui", filepath.Join(installerTempPath, "gui"))
	gui, guiError := NewGui(installerTempPath, translator)
	if guiError != nil {
		log.Println("Unable to create window:", guiError)
	} else {
		gui.Run()
	}
	// }
	// if *cli || guiError != nil {
	// 	tui, err := NewTui(installerTempPath, translator)
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		tui.run()
	// 	}
	// }
}
