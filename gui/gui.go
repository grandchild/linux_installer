package main

import (
	// this is the installer package name - here it refers to the parent directory
	"github.com/grandchild/linux_installer"

	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unsafe"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var gui *Gui

type (
	// EventHandler defines a function to be called when a certain event, identified by
	// a string key, is emitted from the GTK GUI or manually.
	EventHandler map[string]interface{}
	// ScreenHandler is a name and a set of functions that corresponds to a specific
	// screen in the installer process.
	//
	// before() is called immediately after the screen is presented, and allows
	// initializing some GUI elements or running other setup code.
	//
	// after() is called immediately before the next (or previous) screen is presented,
	// and is useful for applying the choices that were made in the still-current
	// screen.
	//
	// undo() is called if the user navigated backwards. It is called immediately after
	// after(). Undo must return a bool: if false is returned then screen switching is
	// aborted. Useful for staying on the same screen, while the undo operation is in
	// progress.
	ScreenHandler struct {
		name   string
		before func()
		after  func()
		undo   func() bool // if undo returns false, then screen switching is canceled.
	}
	// Screen is a single step in the installer, such as the license screen, or
	// selecting the installation path. It also contains its handler configuration.
	Screen struct {
		name    string
		widget  *gtk.Box
		handler ScreenHandler
	}
	// Gui defines a GTK3-based graphical interface for the installer in which a
	// procession of screens must be followed, and includes the actual installation.
	Gui struct {
		installer        *linux_installer.Installer
		builder          *gtk.Builder
		win              *gtk.Window
		content          *gtk.Stack
		backButton       *gtk.Button
		nextButton       *gtk.Button
		quitButton       *gtk.Button
		dirPathEdit      *gtk.Entry
		progressBar      *gtk.Entry
		quitDialog       *gtk.Dialog
		licenseBuf       *gtk.TextBuffer
		curScreen        int
		screenNames      []string
		screens          []Screen
		screenChangeLock sync.Mutex
		translator       *linux_installer.Translator
		config           *linux_installer.Config
	}
)

const displayKey = "_language_display"

// guiEventHandler returns an EventHandler that handles events from the GTK3 elements in
// the .glade GUI-definition file, such as buttons, entries and window events.
func guiEventHandler(g *Gui) (handler EventHandler) {
	return EventHandler{
		"on_quit_clicked":             func() { g.showQuitDialog() },
		"on_main_close":               func() bool { g.showQuitDialog(); return true },
		"on_back_clicked":             func() { g.prevScreen() },
		"on_next_clicked":             func() { g.nextScreen() },
		"on_quit_no_clicked":          func() { g.quitDialog.Hide() },
		"on_quit_yes_clicked":         func() { gtk.MainQuit() },
		"on_quit_dialog_delete_event": func() bool { g.quitDialog.Hide(); return true },
		"on_path_browse_clicked":      func() { g.browseInstallDir() },
		"on_path_reset_clicked":       func() { g.resetInstallDir() },
		"on_path_entry_changed":       func() { g.checkInstallDir() },
		"on_main_destroy":             func() { gtk.MainQuit() },
	}
}

// internalEventHandler, as opposed to guiEventHandler, returns an EventHandler that
// responds to events emitted by the behavioral GUI code, such as installation_finished,
// or to update the progress bar during installation.
func internalEventHandler(g *Gui) (handler EventHandler) {
	return EventHandler{
		"on_installation_finished": g.showResultScreen,
		"on_undo_finished":         g.prevScreen,
		"update_progressbar":       g.updateProgressbar,
	}
}

// screenHandlers registers a list of ScreenHandlers. See ScreenHandler for details.
// This list defines the behavior of the individual screens, and is the central piece to
// connect GUI interaction with the process of the installation.
func screenHandlers(g *Gui) (handlers []ScreenHandler) {
	return []ScreenHandler{
		{
			name: "language",
			before: func() {
				g.backButton.SetSensitive(false)
				g.setLabel(
					"language-text",
					strings.Join(g.translator.GetAllList("_language_pick_text"), "\n"),
				)
				g.setLanguageOptions("language-choose")
			},
			after: func() {
				g.setLanguage("language-choose")
			},
		},
		{
			name: "welcome",
			before: func() {
				g.backButton.SetSensitive(false)
			},
		},
		{
			name: "license",
			before: func() {
				g.nextButton.SetLabel(g.t("button_license_accept"))
			},
		},
		{
			name: "path",
			before: func() {
				g.nextButton.SetLabel(g.t("button_install"))
				g.nextButton.SetSensitive(false)
				g.resetInstallDir()
				g.checkInstallDir()
			},
		},
		{
			name: "progress",
			before: func() {
				g.backButton.SetLabel(g.t("button_abort"))
				g.nextButton.SetSensitive(false)
				g.installer.PreInstall()
				g.installer.StartInstall()
				glib.IdleAdd(g.installationProgress)
			},
			undo: func() bool {
				g.backButton.SetSensitive(false)
				if !g.installer.Done {
					go g.installer.Rollback()
				}
				return g.installer.Done // wait for installer undo
			},
		},
		{
			name: "success",
			before: func() {
				g.installer.CreateLauncher = !g.config.NoLauncher
				g.installer.PostInstall(
					g.translator.Variables,
					g.translator.GetAllStringsRaw(),
				)
				g.quitButton.SetSensitive(false)
				g.backButton.SetSensitive(false)
				g.nextButton.SetLabel(g.t("button_exit"))
			},
			after: func() {
				if getCheckButton(g.builder, "success-run-checkbox").GetActive() {
					g.installer.ExecInstalled()
				}
				gtk.MainQuit()
			},
		},
		{
			name: "failure",
			before: func() {
				g.quitButton.SetSensitive(false)
				g.backButton.SetSensitive(false)
				g.nextButton.SetLabel(g.t("button_exit"))
			},
			after: func() {
				gtk.MainQuit()
			},
		},
	}
}

// NewGui creates a new installer GUI, given a path to a directory for temporary files
// and a translator for translating message strings. The gui object is stored in the
// global variable "gui", and can then be run with "RunGui()".
func NewGui(
	installerTempPath string,
	installer *linux_installer.Installer,
	translator *linux_installer.Translator,
	config *linux_installer.Config,
) error {
	// glib.InitI18n("installer", filepath.Join(installerTempPath, "strings"))
	err := gtk.InitCheck(nil)
	if err != nil {
		return err
	}
	builder, err := gtk.BuilderNewFromFile(
		filepath.Join(installerTempPath, "gui", "gui.glade"),
	)
	if err != nil {
		return err
	}
	gui = &Gui{
		installer:   installer,
		builder:     builder,
		win:         getWindow(builder, "installer-frame"),
		content:     getStack(builder, "content"),
		backButton:  getButton(builder, "button-back"),
		nextButton:  getButton(builder, "button-next"),
		quitButton:  getButton(builder, "button-quit"),
		dirPathEdit: getEntry(builder, "path-entry"),
		progressBar: getEntry(builder, "progress-bar"),
		quitDialog:  getDialog(builder, "quit-dialog"),
		licenseBuf:  getTextBuffer(builder, "license-buf"),
		screens:     make([]Screen, 0, len(screenHandlers(nil))),
		curScreen:   0,
		translator:  translator,
		config:      config,
	}
	gui.builder.ConnectSignals(guiEventHandler(gui))
	for signal, handler := range internalEventHandler(gui) {
		glib.SignalNew(signal)
		gui.win.Connect(signal, handler)
	}

	gui.win.SetTitle(gui.t("title"))
	gui.setLabel("header-text", gui.t("header_text"))

	css, err := gtk.CssProviderNew()
	if err == nil {
		gtkScreen := gui.win.GetScreen()
		if gtkScreen != nil {
			css.LoadFromData(config.GuiCss)
			gtk.AddProviderForScreen(
				gtkScreen, css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
			)
		}
	}

	for _, handler := range screenHandlers(gui) {
		gui.screens = append(gui.screens,
			Screen{
				name:    handler.name,
				widget:  getBox(builder, handler.name),
				handler: handler,
			},
		)
	}
	gui.showScreen(0)
	return err
}

// Run presents the GUI and starts the main event loop. When Run returns the application
// is done and should quit.
func RunGui() {
	gui.win.ShowAll()
	getButton(gui.builder, "button-next").GrabFocus()
	gtk.Main()
}

// showQuitDialog shows a dialog window to the user asking them whether they are sure
// about exiting the installation before finishing. The decision is handled in an event
// handler in guiEventHandler ("on_quit_no_clicked" & "on_quit_yes_clicked"). If the
// current screen is "success" or "failure" (i.e. installation has finished) then the
// installer will exit immediately.
func (g *Gui) showQuitDialog() {
	if g.screens[g.curScreen].name == "success" ||
		g.screens[g.curScreen].name == "failure" {
		gtk.MainQuit()
	}
	g.translateAllLabels(getBox(g.builder, "quit-dialog-box"))
	g.quitDialog.ShowAll()
}

// setScreenElementDefaults resets all navigation buttons to their defaults at the start
// of each screen. Some of the buttons might have been altered by the screen handlers on
// the previous screen.
func (g *Gui) setScreenElementDefaults() {
	g.backButton.SetSensitive(true)
	g.nextButton.SetSensitive(true)
	g.quitButton.SetSensitive(true)
	g.backButton.SetLabel(g.t("button_prev"))
	g.nextButton.SetLabel(g.t("button_next"))
	g.quitButton.SetLabel(g.t("button_quit"))
}

func (g *Gui) prevScreen() { g.showScreen(g.curScreen - 1) }
func (g *Gui) nextScreen() { g.showScreen(g.curScreen + 1) }

// showNamedScreen looks up and shows a specific screen by its name. If no screen by
// that name is found, then nothing happens.
func (g *Gui) showNamedScreen(name string) {
	screenNum := int(-1)
	for i, s := range g.screens {
		if s.name == name {
			screenNum = i
			break
		}
	}
	if screenNum >= 0 {
		g.showScreen(screenNum)
	}
}

// showScreen changes the screen and calls all available screen handler functions. The
// num parameter is automatically clamped to the available screen indexes.
func (g *Gui) showScreen(num int) {
	if num >= 0 && num < len(g.screens) {
		g.screenChangeLock.Lock()
		defer g.screenChangeLock.Unlock()
		if num != g.curScreen && g.screens[g.curScreen].handler.after != nil {
			g.screens[g.curScreen].handler.after()
		}
		if num < g.curScreen && g.screens[g.curScreen].handler.undo != nil {
			res := g.screens[g.curScreen].handler.undo()
			if !res {
				return
			}
		}
		g.curScreen = num
		g.content.SetVisibleChild(g.screens[g.curScreen].widget)
		g.setScreenElementDefaults()
		if g.screens[g.curScreen].handler.before != nil {
			g.screens[g.curScreen].handler.before()
		}
	} else {
		g.showScreen(0)
	}
}

// browseInstallDir opens a GTK file chooser and fills the path edit field with the
// results.
func (g *Gui) browseInstallDir() {
	chooser, err := gtk.FileChooserDialogNewWith2Buttons(
		g.t("dir_browse_title"), g.win,
		gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
		g.t("cancel"), gtk.RESPONSE_CANCEL,
		g.t("ok"), gtk.RESPONSE_ACCEPT,
	)
	if err != nil {
		g.setLabel("path-error-text", g.t("err_couldnt_open_install_path_dialog"))
		log.Println(g.t("err_couldnt_open_install_path_dialog"))
		return
	}
	// set some default folder here?
	chooser.SetCurrentFolder(glib.GetHomeDir())
	if gtk.ResponseType(chooser.Run()) == gtk.RESPONSE_ACCEPT {
		g.dirPathEdit.SetText(chooser.GetFilename())
	}
	chooser.Close()
}

// resetInstallDir resets the path edit field to the predefined default path, which is a
// subdirectory within the user's home. The string for the subdirectory may be set in
// the config through "default_install_dir_name".
func (g *Gui) resetInstallDir() {
	g.dirPathEdit.SetText(filepath.Join(
		glib.GetHomeDir(),
		g.translator.Expand(g.config.DefaultInstallDirName),
	))
}

// checkInstallDir is run whenever the path edit content changes, and tests whether the
// path is valid, its parent writable and whether there is enough space on the disk that
// the path is on. In case of errors it prints them to the (usually invisible) error
// label in the GUI.
func (g *Gui) checkInstallDir() {
	g.nextButton.SetSensitive(true)
	dirName, _ := g.dirPathEdit.GetText()
	err := g.installer.CheckSetInstallDir(dirName)
	if err != nil {
		g.setLabel("path-error-text", g.t(err.Error()))
		g.nextButton.SetSensitive(false)
	} else {
		g.setLabel("path-error-text", "")
	}
	g.setLabel("path-space-required", g.installer.SizeString())
	g.setLabel("path-space-available", g.installer.SpaceString())
	if !g.installer.DiskSpaceSufficient() {
		g.setLabel("path-error-text", g.t("path_err_not_enough_space"))
		g.nextButton.SetSensitive(false)
	}
}

// t returns a localized string for the key, and expands any template variables therein.
// Variables are surrounded by double braces and preceded by a dot like this:
// 	{{.var}}
func (g *Gui) t(key string) (localized string) { return g.translator.Get(key) }

// setLabel changes the text on a label with the given labelId to the given string.
func (g *Gui) setLabel(labelId string, content string) error {
	label := getLabel(g.builder, labelId)
	if label == nil {
		return errors.New(fmt.Sprintf("No label '%s'", labelId))
	}
	label.SetLabel(content)
	return nil
}

// setLanguageOptions fills the dropdown selection box with any language that is
// available through the GUI's translator object and selects the default language.
func (g *Gui) setLanguageOptions(chooserId string) error {
	comboBox := getComboBoxText(g.builder, chooserId)
	if comboBox == nil {
		return errors.New(fmt.Sprintf("No Dropdown '%s'", chooserId))
	}
	comboBox.RemoveAll()
	langList := g.translator.GetLanguages()
	defaultLang := g.translator.GetLanguage()
	displayStrings := g.translator.GetAll(displayKey)
	for _, id := range langList {
		comboBox.Append(id, displayStrings[id])
		if id == defaultLang {
			comboBox.SetActiveID(id)
		}
	}
	return nil
}

// setLanguage gets the currently selected language in the dropdown selection, and
// applies it to all texts throughout the GUI, on all screens.
func (g *Gui) setLanguage(chooserId string) error {
	comboBox := getComboBoxText(g.builder, chooserId)
	if comboBox == nil {
		return errors.New(fmt.Sprintf("No Dropdown '%s'", chooserId))
	}
	err := g.translator.SetLanguage(comboBox.GetActiveID())
	if err != nil {
		return err
	}
	g.translateLabel(getLabel(g.builder, "header-text"))
	for _, screen := range g.screens {
		screen.widget.GetChildren().Foreach(g.translateAllLabels)
	}
	g.translateAllLabels(getBox(g.builder, "quit-dialog-box"))

	licenseFile := fmt.Sprintf("licenses/license_%s.txt", g.translator.GetLanguage())
	licenseText, err := linux_installer.GetResource(licenseFile)
	if err != nil {
		log.Println(fmt.Sprintf("License file not found: %s", licenseFile))
		return err
	}
	g.licenseBuf.SetText(licenseText)
	return nil
}

// translateAllLabels searches recursively, starting with the given GTK item, for labels
// or buttons to translate and translates their contents.
func (g *Gui) translateAllLabels(item interface{}) {
	switch widget := item.(type) {
	case *gtk.Box:
		g.translateAllLabels((*gtk.Widget)(unsafe.Pointer(widget)))
	case *gtk.Widget:
		switch name, _ := widget.GetName(); name {
		case "GtkGrid":
			fallthrough
		case "GtkBox":
			box := (*gtk.Box)(unsafe.Pointer(widget))
			box.GetChildren().Foreach(g.translateAllLabels)
		case "GtkLabel":
			label := (*gtk.Label)(unsafe.Pointer(widget))
			g.translateLabel(label)
		case "GtkCheckButton":
			fallthrough
		case "GtkButton":
			button := (*gtk.Button)(unsafe.Pointer(widget))
			g.translateButton(button)
		}
	}
}

// translateLabel searches for a variable (surrounded by "$" like "$variable$") in the
// label's text and replaces it with the variable's string value from the translator.
func (g *Gui) translateLabel(label *gtk.Label) {
	variable := regexp.MustCompile(`\$[a-zA-Z0-9_]+\$`).FindString(label.GetLabel())
	if len(variable) > 2 {
		label.SetLabel(g.t(variable[1 : len(variable)-1]))
	}
}

// translateButton searches for a variable (surrounded by "$" like "$variable$") in the
// button's text and replaces it with the variable's string value from the translator.
func (g *Gui) translateButton(button *gtk.Button) {
	buttonLabel, err := button.GetLabel()
	if err != nil {
		fmt.Println(err)
	}
	variable := regexp.MustCompile(`\$[a-zA-Z0-9_]+\$`).FindString(buttonLabel)
	if len(variable) > 2 {
		button.SetLabel(g.t(variable[1 : len(variable)-1]))
	}
}

// installationProgress is a glib.Idle function that gets called repeatedly unless it
// returns false. During the file copy process this function checks on the status of the
// installer and emits update signals for the progress bar and finish or undo handlers.
func (g *Gui) installationProgress() (repeat bool) {
	status := g.installer.Status
	g.win.Emit("update_progressbar")
	if status.Done {
		g.win.Emit("on_installation_finished")
		return false
	}
	if status.Aborted {
		g.win.Emit("on_undo_finished")
		return false
	}
	return true
}

// updateProgressbar updates the progress bar with the current filename and the
// percentage of bytes copies to disk so far.
func (g *Gui) updateProgressbar() {
	installingFile := g.installer.NextFile()
	if installingFile != nil {
		g.progressBar.SetText(installingFile.Target)
	}
	g.progressBar.SetProgressFraction(g.installer.Progress())
}

// showResultScreen gets called after the file copy process stops, checks on the status
// of the installer, and changes to the appropriate final screen of the installer GUI,
// success or failure.
func (g *Gui) showResultScreen() {
	g.setLabel("failure-error-text", "")
	if g.installer.Error() != nil {
		log.Println(g.installer.Error().Error())
		g.setLabel("failure-error-text", g.installer.Error().Error())
		g.showNamedScreen("failure")
	} else {
		g.showNamedScreen("success")
	}
}
