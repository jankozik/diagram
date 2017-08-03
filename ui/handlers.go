package ui

import (
	"github.com/jroimartin/gocui"
)

type Fn func(*gocui.Gui, *gocui.View) error

type handler struct {
	views	[]string
	key	interface{}
	keyName	string
	help	string
	action 	func(*UI, bool) Fn
}

type handlers []handler

var keyHandlers = &handlers{
	{mainViews, gocui.KeyTab, "Tab", "Next Panel", onNextPanel},
	{mainViews, 0xFF, "Shift+Tab", "Previous Panel", nil},
	{nil, gocui.KeyCtrlC, "Ctrl+c", "Quit", onQuit},
}

func onNextPanel(ui *UI, wrap bool) Fn {
	return func(*gocui.Gui, *gocui.View) error {
		return ui.NextView(wrap)
	}
}

func onPrevPanel(ui *UI, wrap bool) Fn {
	return func(*gocui.Gui, *gocui.View) error {
		return ui.PrevView(wrap)
	}
}

func onQuit(ui *UI, wrap bool) Fn {
	return func(*gocui.Gui, *gocui.View) error {
		return gocui.ErrQuit
	}
}

func (handlers handlers) ApplyKeyBindings(ui *UI, g *gocui.Gui) error {
	for _, h := range handlers {
		if len(h.views) == 0 {
			h.views = []string{""}
		}
		if h.action == nil {
			continue
		}

		for _, view := range h.views {
			if err := g.SetKeybinding(view, h.key, gocui.ModNone, h.action(ui, true)); err != nil {
				return err
			}
		}
	}
	return nil
}