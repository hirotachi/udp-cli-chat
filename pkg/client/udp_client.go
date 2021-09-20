package client

import (
	"fmt"
	"github.com/rivo/tview"
)

func NewUDPClient() (*tview.Application, error) {
	app := tview.NewApplication()
	connection, err := NewConnection(":5000", "tester")
	if err != nil {
		return nil, err
	}
	box := tview.NewTextView().SetChangedFunc(func() {
		app.Draw()
	})
	box.SetDynamicColors(true).SetScrollable(true).SetBorder(true).SetTitle("Hello, world!")

	go func() {
		history := <-connection.HistoryChan
		box.SetText(fmt.Sprintf("history loaded %d messages", len(history)))
	}()

	go func() {
		for err := range connection.LogChan {
			fmt.Fprint(box, "[red]error:[::-] ", err.Error())
		}
	}()

	app.SetRoot(box, true)
	return app, nil
}
