package linux_installer

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func emptyGtkContainer(container interface {
	GetChildren() *glib.List
	Remove(gtk.IWidget)
}) {
	children := container.GetChildren()
	children.Foreach(func(child interface{}) {
		container.Remove(child.(gtk.IWidget))
	})
}

func getObject(builder *gtk.Builder, name string) glib.IObject {
	obj, err := builder.GetObject(name)
	if err != nil {
		return nil
	}
	return obj
}

func getWindow(builder *gtk.Builder, name string) *gtk.Window {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Window); ok {
		return w
	} else {
		return nil
	}
}

func getDialog(builder *gtk.Builder, name string) *gtk.Dialog {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Dialog); ok {
		return w
	} else {
		return nil
	}
}

func getBox(builder *gtk.Builder, name string) *gtk.Box {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Box); ok {
		return w
	} else {
		return nil
	}
}

func getBoxList(builder *gtk.Builder, names []string) []*gtk.Box {
	boxes := []*gtk.Box{}
	for _, n := range names {
		boxes = append(boxes, getBox(builder, n))
	}
	return boxes
}

func getListBox(builder *gtk.Builder, name string) *gtk.ListBox {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.ListBox); ok {
		return w
	} else {
		return nil
	}
}

func getListBoxRow(builder *gtk.Builder, name string) *gtk.ListBoxRow {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.ListBoxRow); ok {
		return w
	} else {
		return nil
	}
}

func getInfoBar(builder *gtk.Builder, name string) *gtk.InfoBar {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.InfoBar); ok {
		return w
	} else {
		return nil
	}
}

func getLabel(builder *gtk.Builder, name string) *gtk.Label {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Label); ok {
		return w
	} else {
		return nil
	}
}

func getComboBoxText(builder *gtk.Builder, name string) *gtk.ComboBoxText {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.ComboBoxText); ok {
		return w
	} else {
		return nil
	}
}

func getButton(builder *gtk.Builder, name string) *gtk.Button {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Button); ok {
		return w
	} else {
		return nil
	}
}

func getEntry(builder *gtk.Builder, name string) *gtk.Entry {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Entry); ok {
		return w
	} else {
		return nil
	}
}

func getProgressBar(builder *gtk.Builder, name string) *gtk.ProgressBar {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.ProgressBar); ok {
		return w
	} else {
		return nil
	}
}

func getStack(builder *gtk.Builder, name string) *gtk.Stack {
	obj := getObject(builder, name)
	if w, ok := obj.(*gtk.Stack); ok {
		return w
	} else {
		return nil
	}
}
