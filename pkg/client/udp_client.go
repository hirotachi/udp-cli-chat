package client

import (
	"github.com/rivo/tview"
)

func NewUDPClient() (*tview.Application, error) {
	app := tview.NewApplication()
	connection, err := NewConnection(":5000", "tester")
	if err != nil {
		return nil, err
	}
	messageBoard := NewMessageBoard(app, connection)

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.AddItem(messageBoard.Frame, 0, 1, false)

	app.SetRoot(mainFlex, true)
	return app, nil
}
