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

func Interface() {
	logfile := startLogging("installer.log")
	defer logfile.Close()

	openBoxes()
	config, err := ConfigNew()
	if err != nil {
		return
	}
	translator := TranslatorVarNew(config)

	// cli := flag.Bool("cli", false, translator.Get("cli_help_nogui"))
	target := flag.String("target", "", translator.Get("cli_help_target"))
	showLicence := flag.Bool("show-licence", false, translator.Get("cli_help_showlicence"))
	acceptLicence := flag.Bool("accept-licence", false, translator.Get("cli_help_acceptlicence"))
	lang := flag.String("lang", "", translator.Get("cli_help_lang")+" "+strings.Join(translator.GetLanguageOptionKeys(), ", "))
	flag.Parse()

	if len(*lang) > 0 {
		err := translator.SetLanguage(*lang)
		if err != nil {
			fmt.Printf("Language '%s' not available", *lang)
		}
	}
	if *showLicence {
		licenceFile, err := GetResource(fmt.Sprintf("%s_licence.txt", translator.language))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Print(licenceFile)
		}
		return
	}

	installerTempPath := filepath.Join(os.TempDir(), "linux_installer")
	fmt.Println(installerTempPath)
	if len(*target) > 0 {
		if *acceptLicence {
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
			fmt.Println(translator.Get("err_cli_mustacceptlicence"))
		}
		return
	}

	var guiError error
	defer os.RemoveAll(installerTempPath)
	// if !*cli {
	UnpackResourceDir("gui", filepath.Join(installerTempPath, "gui"))
	gui, guiError := GuiNew(installerTempPath, translator)
	if guiError != nil {
		log.Println("Unable to create window:", guiError)
	} else {
		gui.run()
	}
	// }
	// if *cli || guiError != nil {
	// 	tui, err := TuiNew(installerTempPath, translator)
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		tui.run()
	// 	}
	// }
}
