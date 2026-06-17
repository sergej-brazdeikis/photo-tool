package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func collectButtonsForJourney(o fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Button:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func collectSelectWidgetsForJourney(o fyne.CanvasObject) []*widget.Select {
	var out []*widget.Select
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Select:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func collectListsForJourney(o fyne.CanvasObject) []*widget.List {
	var out []*widget.List
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.List:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func firstRadioGroupForJourney(root fyne.CanvasObject) *widget.RadioGroup {
	var walk func(fyne.CanvasObject) *widget.RadioGroup
	walk = func(x fyne.CanvasObject) *widget.RadioGroup {
		if x == nil {
			return nil
		}
		switch v := x.(type) {
		case *widget.RadioGroup:
			return v
		case *container.Scroll:
			return walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				if rg := walk(ch); rg != nil {
					return rg
				}
			}
		}
		return nil
	}
	return walk(root)
}

func collectEntriesForJourney(o fyne.CanvasObject) []*widget.Entry {
	var out []*widget.Entry
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Entry:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *widget.Accordion:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Detail)
				}
			}
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func collectLabelsDeepForJourney(o fyne.CanvasObject) []*widget.Label {
	var out []*widget.Label
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Label:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *widget.Accordion:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Detail)
				}
			}
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

// collectButtonsDeep is shared by journey capture and tests (overlays, accordion).
func collectButtonsDeep(o fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Button:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *widget.PopUp:
			walk(v.Content)
		case *widget.Accordion:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Detail)
				}
			}
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}
