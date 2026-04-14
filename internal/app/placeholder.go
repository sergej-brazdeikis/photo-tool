package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewSectionPlaceholder is a non-deceptive placeholder (AC4): no grids or fake data.
func NewSectionPlaceholder(title, detail string) fyne.CanvasObject {
	head := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := widget.NewLabel(detail)
	body.Wrapping = fyne.TextWrapWord
	return container.NewPadded(container.NewVBox(head, body))
}
