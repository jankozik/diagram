package ui

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/esimov/diagram/canvas"
	"github.com/esimov/diagram/io"
	"github.com/esimov/diagram/version"
	"github.com/fogleman/imview"
	"github.com/jroimartin/gocui"
)

type panelProperties struct {
	title    string
	text     string
	x1       float64
	y1       float64
	x2       float64
	y2       float64
	editable bool
	cursor   bool
	editor   *UI
}

const (
	// Panel constants
	LOGO_PANEL           = "logo"
	SAVED_DIAGRAMS_PANEL = "saved_diagrams"
	LOG_PANEL            = "log"
	DIAGRAM_PANEL        = "diagram"
	PROGRESS_PANEL       = "progress"
	HELP_PANEL           = "help"
	SAVE_MODAL           = "save_modal"
	PROGRESS_MODAL       = "progress_modal"

	// Log messages
	ERROR_EMPTY  = "The editor should not be empty!"
	DIAGRAMS_DIR = "/diagrams"
)

// Main views
var panelViews = map[string]panelProperties{
	LOGO_PANEL: {
		title:    "Diagram",
		text:     version.DrawLogo(),
		x1:       0.0,
		y1:       0.0,
		x2:       0.4,
		y2:       0.25,
		editable: true,
		cursor:   false,
	},
	SAVED_DIAGRAMS_PANEL: {
		title:    "Saved Diagrams",
		text:     "",
		x1:       0.0,
		y1:       0.25,
		x2:       0.4,
		y2:       0.90,
		editable: true,
		cursor:   false,
	},
	LOG_PANEL: {
		title:    "Console",
		text:     "",
		x1:       0.0,
		y1:       0.90,
		x2:       0.4,
		y2:       1.0,
		editable: true,
		cursor:   false,
	},
	DIAGRAM_PANEL: {
		title:    "Editor",
		text:     string(io.ReadFile("sample.txt")),
		x1:       0.4,
		y1:       0.0,
		x2:       1.0,
		y2:       1.0,
		editable: true,
		cursor:   true,
	},
	PROGRESS_PANEL: {
		title:    "Progress",
		text:     "",
		x1:       0.0,
		y1:       0.7,
		x2:       1,
		y2:       0.8,
		editable: false,
		cursor:   false,
	},
}

// Modal views
var modalViews = map[string]panelProperties{
	HELP_PANEL: {
		title:    "Key Shortcuts",
		text:     "",
		editable: false,
	},
	SAVE_MODAL: {
		title:    "Save diagram",
		text:     ".txt",
		editable: true,
	},
	PROGRESS_MODAL: {
		title:    "",
		text:     "\tGenerating...",
		editable: false,
	},
}

var (
	// Panel Views
	mainViews = []string{
		LOGO_PANEL,
		SAVED_DIAGRAMS_PANEL,
		LOG_PANEL,
		DIAGRAM_PANEL,
	}
	modalElements = []string{"save_modal", "save", "cancel"}
	currentFile   string
)

// Initialize the panel views and associate the key bindings to them.
func (ui *UI) Layout(g *gocui.Gui) error {
	initPanel := func(g *gocui.Gui, v *gocui.View) error {
		// Disable panel views selection with mouse in case the modal is activated
		if ui.currentModal == "" {
			cx, cy := v.Cursor()
			line, err := v.Line(cy)
			if err != nil {
				ui.cursors.Restore(v)
				ui.setPanelView(v.Name())
			}

			if cx > len(line) {
				v.SetCursor(ui.cursors.Get(v.Name()))
				ui.cursors.Set(v.Name(), ui.getViewRowCount(v, cy), cy)
			}
			ui.currentView = ui.findViewByName(v.Name())
			ui.setPanelView(v.Name())
			view := panelViews[v.Name()]
			ui.gui.Cursor = view.cursor
		}

		// Refresh the diagram panel with the new diagram content
		cv := ui.gui.CurrentView()
		if cv.Name() == SAVED_DIAGRAMS_PANEL {
			ui.modifyView(DIAGRAM_PANEL)
		}
		return nil
	}

	for _, view := range mainViews {
		if err := g.SetKeybinding(view, gocui.MouseLeft, gocui.ModNone, initPanel); err != nil {
			return err
		}

		if err := g.SetKeybinding(view, gocui.MouseRelease, gocui.ModNone, initPanel); err != nil {
			return err
		}
		if _, err := ui.initPanelView(view); err != nil {
			return err
		}
	}

	// Activate the first panel on first run.
	if v := ui.gui.CurrentView(); v == nil {
		_, err := ui.gui.SetCurrentView(DIAGRAM_PANEL)
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	if err := g.SetKeybinding(DIAGRAM_PANEL, gocui.MouseWheelDown, gocui.ModNone, ui.scrollDown); err != nil {
		return err
	}

	return nil
}

// Scroll down event
func (ui *UI) scrollDown(g *gocui.Gui, v *gocui.View) error {
	maxY := strings.Count(v.Buffer(), "\n")
	if maxY < 1 {
		v.SetCursor(0, 0)
	}
	return nil
}

// Toggle the help view on key pressing.
func (ui *UI) toggleHelp(g *gocui.Gui, content string) error {
	if err := ui.closeOpenedModals(modalElements); err != nil {
		return err
	}
	panelHeight := strings.Count(content, "\n")
	if ui.currentModal == HELP_PANEL {
		ui.gui.DeleteKeybinding("", gocui.MouseLeft, gocui.ModNone)
		ui.gui.DeleteKeybinding("", gocui.MouseRelease, gocui.ModNone)

		// Stop modal timer from firing in case the modal was closed manually.
		// This is needed to prevent the modal being closed before the predefined delay.
		if ui.modalTimer != nil {
			ui.modalTimer.Stop()
		}
		return ui.closeModal(ui.currentModal)
	}
	v, err := ui.openModal(HELP_PANEL, 40, panelHeight, true)
	if err != nil {
		return err
	}
	ui.gui.Cursor = false
	v.Editor = newEditor(ui, &staticViewEditor{})

	fmt.Fprintf(v, content)
	return nil
}

// Create and open the modal window. If "autoHide" parameter is true, the modal will be automatically closed after 5 seconds.
func (ui *UI) openModal(name string, w, h int, autoHide bool) (*gocui.View, error) {
	v, err := ui.createModal(name, w, h)
	if err != nil {
		return nil, err
	}

	if err := ui.setPanelView(name); err != nil {
		return nil, err
	}
	ui.currentModal = name

	if autoHide {
		// Close the modal automatically after 10 seconds
		ui.modalTimer = time.AfterFunc(10*time.Second, func() {
			ui.gui.Update(func(*gocui.Gui) error {
				if err := ui.closeModal(name); err != nil {
					return err
				}
				return nil
			})
		})
	}
	return v, nil
}

// Close the modal window and restore the focus to the last accessed panel view.
func (ui *UI) closeModal(modals ...string) error {
	for _, name := range modals {
		if _, err := ui.gui.View(name); err != nil {
			if err == gocui.ErrUnknownView {
				return nil
			}
			return err
		}
		ui.gui.DeleteView(name)
		ui.gui.DeleteKeybindings(name)
		ui.gui.Cursor = true
		ui.currentModal = ""
	}
	return ui.activatePanelView(ui.currentView)
}

// Initialize and create the modal view.
func (ui *UI) createModal(name string, w, h int) (*gocui.View, error) {
	width, height := ui.gui.Size()
	x1, y1 := width/2-w/2, int(math.Ceil(float64(height/2-h/2-1)))
	x2, y2 := width/2+w/2, int(math.Ceil(float64(height/2+h/2+1)))

	return ui.createModalView(name, x1, y1, x2, y2)
}

// Initialize the panel view.
func (ui *UI) initPanelView(name string) (*gocui.View, error) {
	maxX, maxY := ui.gui.Size()

	p := panelViews[name]

	x1 := int(p.x1 * float64(maxX))
	y1 := int(p.y1 * float64(maxY))
	x2 := int(p.x2*float64(maxX)) - 1
	y2 := int(p.y2*float64(maxY)) - 1

	return ui.createPanelView(name, x1, y1, x2, y2)
}

// Creates the panel view.
func (ui *UI) createPanelView(name string, x1, y1, x2, y2 int) (*gocui.View, error) {
	v, err := ui.gui.SetView(name, x1, y1, x2, y2)
	if err != gocui.ErrUnknownView {
		return nil, err
	}

	p := panelViews[name]
	v.Title = p.title
	v.Editable = p.editable

	if err := ui.writeContent(name, p.text); err != nil {
		return nil, err
	}

	switch name {
	case DIAGRAM_PANEL:
		v.Highlight = false
		v.Autoscroll = true
		v.Editor = newEditor(ui, nil)
	case SAVED_DIAGRAMS_PANEL:
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Editor = newEditor(ui, &staticViewEditor{})

		// Update diagrams directory list
		ui.updateDiagramList(name)
	default:
		v.Editor = newEditor(ui, &staticViewEditor{})
	}
	return v, nil
}

// Creates the modal view.
func (ui *UI) createModalView(name string, x1, y1, x2, y2 int) (*gocui.View, error) {
	v, err := ui.gui.SetView(name, x1, y1, x2, y2)
	if err != gocui.ErrUnknownView {
		return nil, err
	}
	m := modalViews[name]

	v.Title = m.title
	v.Editable = m.editable

	if err := ui.writeContent(name, m.text); err != nil {
		return nil, err
	}

	return v, nil
}

// Activate the view with the id in parameters.
func (ui *UI) activatePanelView(id int) error {
	if err := ui.setPanelView(mainViews[id]); err != nil {
		return err
	}
	v := panelViews[mainViews[id]]
	ui.gui.Cursor = v.cursor
	ui.currentView = id

	return nil
}

// Activate the panel view.
func (ui *UI) setPanelView(name string) error {
	if err := ui.closeModal(ui.currentModal); err != nil {
		return err
	}
	// Save cursor position before switch view
	view := ui.gui.CurrentView()
	x, y := view.Cursor()
	ui.cursors.Set(view.Name(), x, y)

	if _, err := ui.gui.SetCurrentView(name); err != nil {
		if err == gocui.ErrUnknownView {
			return nil
		}
		return err
	}
	return nil
}

// Writes the content into the specific view and set the cursor to the buffer end.
func (ui *UI) writeContent(name, text string) error {
	v, err := ui.gui.View(name)
	if err != nil {
		return err
	}
	v.Clear()
	fmt.Fprintf(v, text)
	v.SetCursor(len(text), 0)
	ui.cursors.Set(name, len(text), 0)

	return nil
}

// Find the view defined by name. Will return the view slice index.
func (ui *UI) findViewByName(name string) int {
	var viewId int = -1
	for idx, v := range mainViews {
		if v == name {
			viewId = idx
			break
		}
	}
	return viewId
}

// Save the diagram content.
func (ui *UI) saveDiagram(name string) error {
	v, err := ui.gui.View(name)
	if err != nil {
		return err
	}

	// Reset log timer firing in case of new incoming message.
	if ui.logTimer != nil {
		ui.logTimer.Stop()
	}

	if len(v.ViewBuffer()) == 0 {
		ui.consoleLog = ERROR_EMPTY
		if err := ui.log(ui.consoleLog, true); err != nil {
			return err
		}
	}
	return ui.showSaveModal(SAVE_MODAL)
}

// ASCII -> to PNG converter.
func (ui *UI) drawDiagram(name string) error {
	var output string

	v, err := ui.gui.View(name)
	if err != nil {
		return err
	}

	if len(v.ViewBuffer()) == 0 {
		ui.consoleLog = ERROR_EMPTY
		if err := ui.log(ui.consoleLog, true); err != nil {
			return err
		}
	}
	if currentFile == "" {
		output = "output.png"
	} else {
		output = strings.TrimSuffix(currentFile, ".txt")
		output = output + ".png"
	}
	// Show progress
	ui.showProgressModal(PROGRESS_MODAL)

	cwd, err := filepath.Abs(filepath.Dir(""))
	if err != nil {
		log.Fatal(err)
	}
	filePath := cwd + "/output/"

	// Create output directory in case it does not exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		os.Mkdir(filePath, os.ModePerm)
	}

	// Generate the hand-drawn diagram.
	err = canvas.DrawDiagram(v.Buffer(), filePath+output, ui.fontpath)
	if err == nil {
		ui.log(fmt.Sprintf("Successfully converted the ascii diagram into %s!", output), false)
	} else {
		ui.log("Error on saving the ascii diagram! Please check if the output folder exists.", true)
	}

	// Close progress modal after 1 second
	ui.modalTimer = time.AfterFunc(1*time.Second, func() {
		ui.gui.Update(func(*gocui.Gui) error {
			ui.nextItem = 0 // reset modal elements counter to 0
			if err := ui.closeModal(PROGRESS_MODAL); err != nil {
				return err
			}
			defer func() {
				image, _ := imview.LoadImage(filePath + output)
				view := imview.ImageToRGBA(image)
				imview.Show(view)
			}()
			return nil
		})
	})
	return nil
}

// Show the save modal.
func (ui *UI) showSaveModal(name string) error {
	var saveBtn, cancelBtn *gocui.View

	if err := ui.closeModal(ui.currentModal); err != nil {
		return err
	}
	modal, err := ui.openModal(name, 40, 4, false)
	if err != nil {
		return err
	}
	if ui.modalTimer != nil {
		ui.modalTimer.Stop()
	}

	ui.gui.Cursor = true
	modal.Editor = newEditor(ui, &modalSaveEditor{30})
	modal.SetCursor(0, 0)

	ui.gui.DeleteKeybinding("", gocui.MouseLeft, gocui.ModNone)
	ui.gui.DeleteKeybinding("", gocui.MouseRelease, gocui.ModNone)

	// Close event handler
	onClose := func(*gocui.Gui, *gocui.View) error {
		ui.nextItem = 0 // reset modal elements counter to 0
		if err := ui.closeOpenedModals(modalElements); err != nil {
			return err
		}
		return nil
	}

	// Save event handler
	onSave := func(*gocui.Gui, *gocui.View) error {
		diagram, _ := ui.gui.View(DIAGRAM_PANEL)
		v := modalViews[name]

		ui.nextItem = 0 // reset modal elements counter to 0

		// Check if the file name contains only letters, numbers and underscores.
		buffer := strings.TrimSpace(strings.Replace(modal.ViewBuffer(), v.text, "", -1))
		re := regexp.MustCompile("^[a-zA-Z0-9_]*$")
		res := re.MatchString(buffer)

		if len(diagram.ViewBuffer()) == 0 {
			ui.log("The diagram is empty!", true)
			return nil
		}

		if len(strings.TrimSpace(modal.Buffer())) <= len(v.text) {
			ui.log("File name should not be empty!", true)
		} else if res {
			file := buffer + v.text
			_, err := io.SaveFile(file, DIAGRAMS_DIR, diagram.ViewBuffer())
			if err != nil {
				return err
			}
			ui.log(fmt.Sprintf("The file has been saved as: %s", file), false)
		} else {
			ui.log("File name should contain only letters, numbers and underscores!", true)
		}

		if err := ui.closeOpenedModals(modalElements); err != nil {
			return err
		}

		// Update diagrams directory list
		ui.updateDiagramList(SAVED_DIAGRAMS_PANEL)

		return nil
	}

	// Tab event handler
	onNext := func(*gocui.Gui, *gocui.View) error {
		var pv *gocui.View

		if err := ui.nextElement(modalElements); err != nil {
			return err
		}
		if (ui.nextItem - 1) > 0 {
			pv, _ = ui.gui.View(modalElements[ui.nextItem-1])
		} else {
			pv, _ = ui.gui.View(modalElements[len(modalElements)-1])
		}
		pv.Highlight = false
		if ui.nextItem == 0 {
			ui.gui.Cursor = true
		}
		return nil
	}

	// Get modal with and height
	sw, sh := ui.gui.Size()
	mw, _ := modal.Size()

	saveBtn, err = ui.createButtonWidget("save", sw/2-mw/2, sh/2, "Save", nil)
	if err != nil {
		return err
	}

	if saveBtn != nil {
		saveBtnSize, _ := saveBtn.Size()
		//Calculate the current modal button position relative to the previous button.
		cancelBtn, err = ui.createButtonWidget("cancel", (sw/2-mw/2)+saveBtnSize+4, sh/2, "Cancel", nil)
		if err != nil {
			return err
		}
		if err := ui.gui.SetKeybinding(saveBtn.Name(), gocui.KeyEnter, gocui.ModNone, onSave); err != nil {
			return err
		}
		if err := ui.gui.SetKeybinding(cancelBtn.Name(), gocui.KeyEnter, gocui.ModNone, onClose); err != nil {
			return err
		}
	}

	keys := []gocui.Key{gocui.KeyCtrlS, gocui.KeyEnter}
	for _, k := range keys {
		if err := ui.gui.SetKeybinding(name, k, gocui.ModNone, onSave); err != nil {
			return err
		}
	}
	// Associate the close modal key binding to each modal element.
	for _, view := range modalElements {
		if err := ui.gui.SetKeybinding(view, gocui.KeyCtrlX, gocui.ModNone, onClose); err != nil {
			return err
		}
		if err := ui.gui.SetKeybinding(view, gocui.KeyTab, gocui.ModNone, onNext); err != nil {
			return err
		}
	}

	// Hide log message after 4 seconds
	ui.logTimer = time.AfterFunc(4*time.Second, func() {
		ui.gui.Update(func(*gocui.Gui) error {
			ui.clearLog()
			return nil
		})
	})

	return nil
}

// Show progress modal.
func (ui *UI) showProgressModal(name string) error {
	if err := ui.closeModal(ui.currentModal); err != nil {
		return err
	}
	_, err := ui.openModal(name, 40, 1, false)
	if err != nil {
		return err
	}
	if ui.modalTimer != nil {
		ui.modalTimer.Stop()
	}

	ui.gui.DeleteKeybinding("", gocui.MouseLeft, gocui.ModNone)
	ui.gui.DeleteKeybinding("", gocui.MouseRelease, gocui.ModNone)

	return nil
}

// updateView update the view content
func (ui *UI) updateView(v *gocui.View, buffer string) error {
	if v != nil {
		v.Clear()
		if err := ui.writeContent(v.Name(), buffer); err != nil {
			return err
		}
	}
	return nil
}

// modifyView will change the editor content with the content of the opened file.
func (ui *UI) modifyView(name string) error {
	v, err := ui.gui.View(name)
	if err != nil {
		return err
	}
	if v != nil {
		cv, err := ui.gui.View(SAVED_DIAGRAMS_PANEL)
		if err != nil {
			return err
		}
		_, cy := cv.Cursor()
		cwd, err := filepath.Abs(filepath.Dir(""))
		if err != nil {
			log.Fatal(err)
		}
		currentFile = ui.getViewRow(cv, cy)[0]
		buffer := string(io.ReadFile(cwd + "/" + DIAGRAMS_DIR + "/" + currentFile))

		if err := ui.updateView(v, buffer); err != nil {
			return err
		}
	}
	return nil
}

// updateDiagramList updates the diagram panel content.
func (ui *UI) updateDiagramList(name string) error {
	v, err := ui.gui.View(name)
	if err != nil {
		return err
	}
	v.Clear()
	diagrams, _ := io.ListDiagrams(DIAGRAMS_DIR)

	for idx, diagram := range diagrams {
		if idx < len(diagrams)-1 {
			fmt.Fprintf(v, diagram+"\n")
		} else {
			fmt.Fprintf(v, diagram)
		}
		v.SetCursor(len(diagram), 0)
		ui.cursors.Set(name, len(diagram), 0)
	}
	return nil
}

// closeOpenedModals will close all the opened modal elements.
func (ui *UI) closeOpenedModals(views []string) error {
	for _, v := range views {
		if view, _ := ui.gui.View(v); view != nil {
			ui.closeModal(view.Name())
		}
	}
	return nil
}

// nextView activate the next panel.
func (ui *UI) nextView(wrap bool) error {
	var index int
	index = ui.currentView + 1
	if index > len(mainViews)-1 {
		if wrap {
			index = 0
		} else {
			return nil
		}
	}
	ui.currentView = index % len(mainViews)
	return ui.activatePanelView(ui.currentView)
}

// prevView activate the previous panel.
func (ui *UI) prevView(wrap bool) error {
	var index int
	index = ui.currentView - 1
	if index < 0 {
		if wrap {
			index = len(mainViews) - 1
		} else {
			return nil
		}
	}
	ui.currentView = index % len(mainViews)
	return ui.activatePanelView(ui.currentView)
}

// ClearView clears the panel view.
func (ui *UI) ClearView(name string) {
	v, _ := ui.gui.View(name)
	v.Clear()
}

// DeleteView deletes the current view.
func (ui *UI) DeleteView(name string) {
	v, _ := ui.gui.View(name)
	ui.gui.DeleteView(v.Name())
}
