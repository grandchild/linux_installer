// +build ignore
package linux_installer

import (
	"errors"
	"fmt"
	"log"
	// "os"
	// "path/filepath"
	// "reflect"

	gotui "github.com/marcusolsson/tui-go"
	"gopkg.in/yaml.v2"
)

type (
	tuiStep struct {
		name   string
		text   string
		before func()
		after  func()
		undo   func() bool
	}
	TuiDefinitionElement struct {
		Id       string                 `yaml:"id"`
		Typ      string                 `yaml:"type"`
		Text     string                 `yaml:"text,omitempty"`
		Title    string                 `yaml:"title,omitempty"`
		Cols     int                    `yaml:"cols"`
		Rows     int                    `yaml:"rows"`
		Max      int                    `yaml:"max"`
		X        int                    `yaml:"x"`
		Y        int                    `yaml:"y"`
		Border   bool                   `yaml:"border"`
		Children []TuiDefinitionElement `yaml:"children,omitempty"`
	}
	TuiDefinition []TuiDefinitionElement
	TuiBuilder    struct {
		source      string
		definitions TuiDefinition
		widgets     map[string]*gotui.Widget
		roots       []gotui.Widget
	}
	Tui struct {
		builder         TuiBuilder
		installer       Installer
		curStep         int
		StepNames       []string
		Steps           []tuiStep
		translator      Translator
		stringVariables map[string]string
		ui              gotui.UI
		curProgress     int
	}
)

var gotuiTypes = map[string]interface{}{
	"Button":     gotui.NewButton,
	"Entry":      gotui.NewEntry,
	"Grid":       gotui.NewGrid,
	"Label":      gotui.NewLabel,
	"List":       gotui.NewList,
	"Padder":     gotui.NewPadder,
	"Progress":   gotui.NewProgress,
	"ScrollArea": gotui.NewScrollArea,
	"Spacer":     gotui.NewSpacer,
	"StatusBar":  gotui.NewStatusBar,
	"Table":      gotui.NewTable,
	"TextEdit":   gotui.NewTextEdit,
}

func tuiSteps() []tuiStep {
	return []tuiStep{
		{name: "language"},
		{name: "welcome"},
		{name: "license"},
		{name: "path"},
		{name: "shortcut"},
		{name: "progress"},
		{name: "final"},
	}
}

func TuiNew(installerTempPath string, translator Translator) (Tui, error) {
	builder, errs := TuiBuilderNew(MustGetResource("tui/tui.yml"))
	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("TUI parse error: %s", err)
		}
		return Tui{}, errors.New("Unable to load tui definition")
	}
	progress := gotui.NewProgress(100)
	progress.SetCurrent(30)
	ui, err := builder.GetUI()
	if err != nil {
		return Tui{}, err
	}

	tui := Tui{
		builder:         builder,
		installer:       InstallerNew(installerTempPath),
		Steps:           tuiSteps(),
		curStep:         0,
		translator:      translator,
		stringVariables: translator.variables,
		ui:              ui,
		curProgress:     30,
	}

	ui.SetKeybinding("Esc", ui.Quit)
	ui.SetKeybinding("q", ui.Quit)
	ui.SetKeybinding("Ctrl+C", ui.Quit)
	ui.SetKeybinding("Enter", tui.nextStep)
	ui.SetKeybinding("Backspace2", tui.prevStep)

	tui.showStep(0)
	tui.setBoxTitle("main_layout", tui.translator.Get("title"))
	return tui, nil
}

func (t *Tui) prevStep() { t.showStep(t.curStep - 1) }
func (t *Tui) nextStep() { t.showStep(t.curStep + 1) }
func (t *Tui) showStep(num int) {
	if num >= 0 && num < len(t.Steps) {
		if num != t.curStep && t.Steps[t.curStep].after != nil {
			t.Steps[t.curStep].after()
		}
		if num < t.curStep && t.Steps[t.curStep].undo != nil {
			res := t.Steps[t.curStep].undo()
			if !res {
				return
			}
		}
		t.curStep = num
		t.replaceContent(num)
		t.setBoxTitle("main_layout", t.translator.Get(t.Steps[num].name+"_header"))
		t.setLabelText("footer", fmt.Sprintf("%d/%d", num+1, len(t.Steps)))
		// t.setStepElementDefaults()
		if t.Steps[t.curStep].before != nil {
			t.Steps[t.curStep].before()
		}
	} else {
		t.showStep(0)
	}

}

func (t *Tui) replaceContent(num int) error {
	if content, ok := t.builder.widgets["content"]; ok {
		if box, ok := (*content).(*gotui.Box); ok {
			box.Remove(0)
			box.Append(t.builder.roots[num+1])
			return nil
		}
		return errors.New("'content' widget doesn't have Box type")
	}
	return errors.New("UI has no element 'content'")
}

func (t *Tui) run() {
	t.ui.Run()
}

func (t *Tui) setBoxTitle(id string, title string) {
	if widget, ok := t.builder.widgets[id]; ok {
		if box, ok := (*widget).(*gotui.Box); ok {
			box.SetTitle(" " + title + " ")
		}
	}
}

func (t *Tui) setLabelText(id string, text string) {
	if widget, ok := t.builder.widgets[id]; ok {
		if box, ok := (*widget).(*gotui.Label); ok {
			box.SetText(text)
		}
	}
}

///
func TuiBuilderNew(source string) (TuiBuilder, []error) {
	var definition TuiDefinition
	errorList := []error{}
	err := yaml.Unmarshal([]byte(source), &definition)
	if err != nil {
		errorList = append(errorList, err)
		return TuiBuilder{}, errorList
	}
	// log.Println(definition, len(definition))
	builder := TuiBuilder{
		source,
		definition,
		map[string]*gotui.Widget{},
		[]gotui.Widget{},
	}
	for _, def := range definition {
		widget, errs := builder.Unmarshal(def)
		errorList = append(errorList, errs...)
		builder.roots = append(builder.roots, widget)
	}
	return builder, errorList
}

func (b *TuiBuilder) GetUI() (gotui.UI, error) {
	return gotui.New(b.roots[0])
}

func (b *TuiBuilder) Unmarshal(element TuiDefinitionElement) (gotui.Widget, []error) {
	errorList := []error{}
	children := []gotui.Widget{}
	switch element.Typ {
	case "VBox", "HBox", "Grid", "Table", "ScrollArea", "Padder":
		for _, childDef := range element.Children {
			child, errs := b.Unmarshal(childDef)
			if len(errs) == 0 {
				children = append(children, child)
			} else {
				errorList = append(errorList, errs...)
			}
		}
	default:
	}

	var widget gotui.Widget
	switch element.Typ {
	case "HBox":
		box := gotui.NewHBox(children...)
		box.SetBorder(element.Border)
		box.SetTitle(element.Title)
		b.register(element.Id, box)
		widget = box
	case "VBox":
		box := gotui.NewVBox(children...)
		box.SetBorder(element.Border)
		box.SetTitle(element.Title)
		b.register(element.Id, box)
		widget = box
	case "Table":
		table := gotui.NewTable(element.Cols, element.Rows)
		b.register(element.Id, table)
		widget = table
	case "ScrollArea":
		area := gotui.NewScrollArea(children[0])
		b.register(element.Id, area)
		widget = area
	case "Spacer":
		widget = gotui.NewSpacer()
		b.register(element.Id, widget)
	case "Padder":
		if len(children) > 0 {
			widget = gotui.NewPadder(element.X, element.Y, children[0])
		} else {
			widget = gotui.NewPadder(element.X, element.Y, nil)
		}
		b.register(element.Id, widget)
	case "Label":
		widget = gotui.NewLabel(element.Text)
		b.register(element.Id, widget)
	case "Button":
		widget = gotui.NewButton(element.Text)
		b.register(element.Id, widget)
	case "Progress":
		progress := gotui.NewProgress(element.Max)
		b.register(element.Id, progress)
		widget = progress
	case "Entry":
		entry := gotui.NewEntry()
		b.register(element.Id, entry)
		widget = entry
	default:
		errorList = append(errorList, errors.New(fmt.Sprintf("No such element type: '%s' (id '%s')", element.Typ, element.Id)))
		widget = nil
	}
	return widget, errorList
}

func (b *TuiBuilder) register(id string, widget gotui.Widget) {
	if widget != nil && id != "" {
		b.widgets[id] = &widget
	}
}
