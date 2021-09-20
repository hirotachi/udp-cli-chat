package client

import (
	"github.com/rivo/tview"
	"github.com/rs/xid"
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
	return app, nil
}
