package linux_installer

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

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
		curScreen        int
		screenNames      []string
		screens          []Screen
		screenChangeLock sync.Mutex
		translator       Translator
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
		"on_path_entry_changed":  func() { g.checkInstallDir() },
		"on_main_close":          func() bool { g.quitDialog.ShowAll(); return true },
		"on_main_destroy":        func() { gtk.MainQuit() },
	}
}

func internalEventHandler(g *Gui) (handler EventHandler) {
	return EventHandler{
		"on_installation_finished": g.nextScreen,
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
				g.setLabel("language_text", strings.Join(g.translator.GetAllVersionsList("_language_pick_text"), "\n"))
				g.setLanguageOptions("language_choose", g.translator.GetLanguage())
			},
			after: func() {
				g.setLanguage("language_choose")
				log.Println("GUI Language:", g.translator.language)
				g.t("string")
			},
		},
		{
			name: "welcome",
			before: func() {
				g.setLabel("h1", g.t("welcome_header"))
				g.setLabel("body1", g.t("welcome_text"))
			},
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
				g.dirPathEdit.SetText(filepath.Join(glib.GetHomeDir(), "linux_installer_test"))
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
			name: "final",
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

func (g *Gui) setScreenElementDefaults() {
	g.backButton.SetSensitive(true)
	g.nextButton.SetSensitive(true)
	g.quitButton.SetSensitive(true)
	g.backButton.SetLabel(g.t("button_prev"))
	g.nextButton.SetLabel(g.t("button_next"))
	g.quitButton.SetLabel(g.t("button_quit"))
}

func GuiNew(resourcesPath string, translator Translator) (Gui, error) {
	// glib.InitI18n("installer", filepath.Join(resourcesPath, "strings"))
	// gtk.Init(nil)
	err := gtk.InitCheck(nil)
	if err != nil {
		return Gui{}, err
	}
	builder, err := gtk.BuilderNewFromFile(filepath.Join(resourcesPath, "gui", "gui_slider.glade"))
	if err != nil {
		return Gui{}, err
	}
	gui := Gui{
		installer:   InstallerNew(),
		builder:     builder,
		win:         getWindow(builder, "installer_frame"),
		content:     getStack(builder, "content"),
		backButton:  getButton(builder, "button_back"),
		nextButton:  getButton(builder, "button_next"),
		quitButton:  getButton(builder, "button_quit"),
		dirPathEdit: getEntry(builder, "path_entry"),
		progressBar: getEntry(builder, "progress_bar"),
		quitDialog:  getDialog(builder, "quit_dialog"),
		screens:     make([]Screen, 0, len(screenHandlers(nil))),
		curScreen:   0,
		translator:  translator,
	}
	gui.builder.ConnectSignals(guiEventHandler(&gui))
	for signal, handler := range internalEventHandler(&gui) {
		glib.SignalNew(signal)
		gui.win.Connect(signal, handler)
	}

	gui.win.SetTitle(gui.translator.Get("title"))
	gui.setLabel("header_title", gui.translator.Get("header"))

	// css, err := gtk.CssProviderNew()
	// if err == nil {
	// 	gtkScreen, err := gui.win.GetScreen()
	// 	if err == nil {
	// 		css.LoadFromData(".main_window { background: #fff; }")
	// 		gtk.AddProviderForScreen(gtkScreen, css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	// 	}
	// }

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
	return gui, nil
}

func (g *Gui) run() {
	g.win.ShowAll()
	gtk.Main()
}

func (g *Gui) prevScreen() { g.showScreen(g.curScreen - 1) }
func (g *Gui) nextScreen() { g.showScreen(g.curScreen + 1) }
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
		// g.content.Remove(g.screens[g.curScreen].widget)
		g.curScreen = num
		g.content.SetVisibleChild(g.screens[g.curScreen].widget)
		// g.content.Add(g.screens[g.curScreen].widget)
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
	// chooser.SetCurrentFolder("~")
	if gtk.ResponseType(chooser.Run()) == gtk.RESPONSE_ACCEPT {
		g.dirPathEdit.SetText(chooser.GetFilename())
	}
	chooser.Close()
}

func (g *Gui) checkInstallDir() {
	dirName, _ := g.dirPathEdit.GetText()
	err := g.installer.CheckInstallDir(dirName)
	if err != nil {
		// TODO: set something to red
		log.Println(err)
	}
	g.nextButton.SetSensitive(true)
}

// t returns a localized string for the key, and expands any template
// variables therein. Variables are surrounded by double braces like this:
//
// 	{{var}}
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
	g.translator.SetLanguage(comboBox.GetActiveID())
	return nil
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
