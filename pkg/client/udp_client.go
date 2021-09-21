package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	MessageView = "message_view"
	InputView   = "input_view"
)

func NewUDPClient() (*tview.Application, error) {
	app := tview.NewApplication()
	connection := NewConnection(app)
	messageBoard := NewMessageBoard(app, connection)
	inputSection := NewInputSection(messageBoard)

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.AddItem(messageBoard.Frame, 0, 1, false)
	mainFlex.AddItem(inputSection.View, 2, 1, false)

	// initial connection form
	serverAddress := ":5000"
	username := "tester"
	form := tview.NewForm().
		AddInputField("Server address", serverAddress, 20, nil, func(text string) {
			serverAddress = text
		}).
		AddInputField("Username", username, 20, nil, func(text string) {
			username = text
		})
	form.AddButton("Connect", func() {
		if err := connection.Connect(serverAddress, username); err != nil {
			form.SetTitle("something went wrong try again").SetTitleColor(tcell.ColorRed)
			return
		}
		app.SetRoot(mainFlex, true)
		app.SetFocus(inputSection.View)
	})
	form.AddButton("Quit", func() {
		app.Stop()
	})
	form.SetTitle("Enter required data")
	form.SetBorder(true)
	form.SetTitleAlign(tview.AlignLeft)

	app.SetRoot(form, true)

	app.SetFocus(form)
	// help focus other views
	focus := func(view string) {
		switch view {
		case MessageView:
			app.SetFocus(messageBoard.Frame)
		case InputView:
			app.SetFocus(inputSection.View)
		}
		app.SetFocus(messageBoard.View)
	}

	messageBoard.Frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.SetFocus(inputSection.View)
			return nil
		}
		return event
	})
	inputSection.Focus = focus
	return app, nil
}
