package client

import "github.com/rivo/tview"

func NewUDPClient() *tview.Application {
	app := tview.NewApplication()
	box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	app.SetRoot(box, true)

	return app
}
