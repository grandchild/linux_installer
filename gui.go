package linux_installer

import (
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

type (
	EventHandler  map[string]interface{}
	ScreenHandler struct {
		name   string
		before func()
		after  func()
		undo   func() bool // undo() should return true. if it returns false, then screen switching is cancelled.
	}
	Screen struct {
		name    string
		widget  *gtk.Box
		handler ScreenHandler
	}
	Gui struct {
		installer        Installer
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
		translator       Translator
		config           *Config
	}
)

func guiEventHandler(g *Gui) (handler EventHandler) {
	return EventHandler{
		"on_quit_clicked":        func() { g.quitDialog.ShowAll(); g.quitDialog.GrabFocus() },
		"on_back_clicked":        func() { g.prevScreen() },
		"on_next_clicked":        func() { g.nextScreen() },
		"on_quit_no_clicked":     func() { g.quitDialog.Hide() },
		"on_quit_yes_clicked":    func() { gtk.MainQuit() },
		"on_path_browse_clicked": func() { g.browseInstallDir() },
		"on_path_reset_clicked":  func() { g.resetInstallDir() },
		"on_path_entry_changed":  func() { g.checkInstallDir() },
		"on_main_destroy":        func() { gtk.MainQuit() },
		"on_main_close": func() bool {
			g.translateAllLabels(getBox(g.builder, "quit-dialog-box"))
			g.quitDialog.ShowAll()
			return true
		},
	}
}

func internalEventHandler(g *Gui) (handler EventHandler) {
	return EventHandler{
		"on_installation_finished": g.showResultScreen,
		"on_undo_finished":         g.prevScreen,
		"update_progressbar":       g.updateProgressbar,
	}
}

func screenHandlers(g *Gui) (handlers []ScreenHandler) {
	return []ScreenHandler{
		{
			name: "language",
			before: func() {
				g.backButton.SetSensitive(false)
				g.setLabel("language-text", strings.Join(g.translator.GetAllVersionsList("_language_pick_text"), "\n"))
				g.setLanguageOptions("language-choose", g.translator.GetLanguage())
			},
			after: func() {
				g.setLanguage("language-choose")
			},
		},
		{
			name: "welcome",
		},
		{
			name: "license",
			before: func() {
				g.nextButton.SetLabel(g.t("license_button_accept"))
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
				g.quitButton.SetSensitive(false)
				g.backButton.SetSensitive(false)
				g.nextButton.SetLabel(g.t("button_exit"))
			},
			after: func() {
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

// NewGui returns a new installer GUI, given a path to a directory for temporary files
// and a translator for translating message strings.
func NewGui(installerTempPath string, translator Translator, config *Config) (*Gui, error) {
	// glib.InitI18n("installer", filepath.Join(installerTempPath, "strings"))
	err := gtk.InitCheck(nil)
	if err != nil {
		return &Gui{}, err
	}
	builder, err := gtk.BuilderNewFromFile(filepath.Join(installerTempPath, "gui", "gui_slider.glade"))
	if err != nil {
		return &Gui{}, err
	}
	gui := Gui{
		installer:   NewInstaller(installerTempPath, config),
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
	gui.builder.ConnectSignals(guiEventHandler(&gui))
	for signal, handler := range internalEventHandler(&gui) {
		glib.SignalNew(signal)
		gui.win.Connect(signal, handler)
	}

	gui.win.SetTitle(gui.t("title"))
	gui.setLabel("header-text", gui.t("header_text"))

	css, err := gtk.CssProviderNew()
	if err == nil {
		gtkScreen, err := gui.win.GetScreen()
		if err == nil {
			css.LoadFromData(config.GuiCss)
			gtk.AddProviderForScreen(gtkScreen, css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
		}
	}

	for _, handler := range screenHandlers(&gui) {
		gui.screens = append(gui.screens,
			Screen{
				name:    handler.name,
				widget:  getBox(builder, handler.name),
				handler: handler,
			},
		)
	}
	gui.showScreen(0)
	return &gui, nil
}

func (g *Gui) Run() {
	g.win.ShowAll()
	gtk.Main()
}

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

func (g *Gui) browseInstallDir() {
	chooser, err := gtk.FileChooserDialogNewWith2Buttons(
		g.t("dir_browse_title"), g.win,
		gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
		g.t("cancel"), gtk.RESPONSE_CANCEL,
		g.t("ok"), gtk.RESPONSE_ACCEPT,
	)
	if err != nil {
		log.Println(g.t("err_couldnt_open_install_path_dialog"))
	}
	// set some default folder here?
	chooser.SetCurrentFolder(glib.GetHomeDir())
	if gtk.ResponseType(chooser.Run()) == gtk.RESPONSE_ACCEPT {
		g.dirPathEdit.SetText(chooser.GetFilename())
	}
	chooser.Close()
}

func (g *Gui) resetInstallDir() {
	g.dirPathEdit.SetText(filepath.Join(
		glib.GetHomeDir(),
		g.translator.Expand(g.config.DefaultInstallDirName),
	))
}

func (g *Gui) checkInstallDir() {
	g.nextButton.SetSensitive(true)
	dirName, _ := g.dirPathEdit.GetText()
	err := g.installer.CheckInstallDir(dirName)
	if err != nil {
		g.setLabel("path-error-text", g.t(err.Error()))
		g.nextButton.SetSensitive(false)
	} else {
		g.setLabel("path-error-text", "")
	}
	g.installer.PrepareDataFiles()
	g.setLabel("path-space-required", g.installer.SizeString())
	g.setLabel("path-space-available", g.installer.SpaceString())
	if !g.installer.DiskSpaceSufficient() {
		g.setLabel("path-error-text", g.t("path_err_not_enough_space"))
		g.nextButton.SetSensitive(false)
	}
}

// t returns a localized string for the key, and expands any template variables therein.
// Variables are surrounded by double braces and preceded by a dot like this:
//
// 	{{.var}}
func (g *Gui) t(key string) (localized string) {
	return g.translator.Get(key)
}

func (g *Gui) setLabel(labelId string, content string) error {
	label := getLabel(g.builder, labelId)
	if label == nil {
		return errors.New(fmt.Sprintf("No label '%s'", labelId))
	}
	label.SetLabel(content)
	return nil
}

func (g *Gui) setLanguageOptions(chooserId string, defaultLang string) error {
	comboBox := getComboBoxText(g.builder, chooserId)
	if comboBox == nil {
		return errors.New(fmt.Sprintf("No Dropdown '%s'", chooserId))
	}
	comboBox.RemoveAll()
	langList := g.translator.GetLanguages()
	displayStrings := g.translator.GetAllVersions("_language_display")
	for _, id := range langList {
		comboBox.Append(id, displayStrings[id])
		if id == defaultLang {
			comboBox.SetActiveID(id)
		}
	}
	return nil
}

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
	licenseText, err := GetResource(licenseFile)
	if err != nil {
		log.Println(fmt.Sprintf("License file not found: %s", licenseFile))
		return err
	}
	g.licenseBuf.SetText(licenseText)
	return nil
}

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
		case "GtkButton":
			button := (*gtk.Button)(unsafe.Pointer(widget))
			g.translateButton(button)
		}
	}
}

func (g *Gui) translateLabel(label *gtk.Label) {
	variable := regexp.MustCompile(`\$[a-zA-Z0-9_]+\$`).FindString(label.GetLabel())
	if len(variable) > 2 {
		label.SetLabel(g.t(variable[1 : len(variable)-1]))
	}
}

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

func (g *Gui) installationProgress() (repeat bool) {
	status := g.installer.Status()
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

func (g *Gui) updateProgressbar() {
	installingFile := g.installer.NextFile()
	if installingFile != nil {
		g.progressBar.SetText(installingFile.Target)
	}
	g.progressBar.SetProgressFraction(g.installer.Progress())
}

func (g *Gui) showResultScreen() {
	if g.installer.err != nil {
		log.Println(g.installer.err.Error())
		g.showNamedScreen("failure")
	} else {
		g.showNamedScreen("success")
	}
}
