package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/xid"
)

const (
	MessageView = "message_view"
	InputView   = "input_view"
)

func NewUDPClient() (*tview.Application, error) {
	app := tview.NewApplication()
	connection, err := NewConnection(":5000", xid.New().String())
	if err != nil {
		return nil, err
	}
	messageBoard := NewMessageBoard(app, connection)
	inputSection := NewInputSection(messageBoard)

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.AddItem(messageBoard.Frame, 0, 1, false)
	mainFlex.AddItem(inputSection.View, 2, 1, false)

	app.SetRoot(mainFlex, true)

	app.SetFocus(inputSection.View)
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
	messageBoard.Focus = focus
	inputSection.Focus = focus
	return app, nil
}
