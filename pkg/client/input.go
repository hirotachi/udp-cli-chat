package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

type InputSection struct {
	View         *tview.InputField
	MessageBoard *MessageBoard
}

func NewInputSection(messageBoard *MessageBoard) *InputSection {
	inputView := tview.NewInputField()
	inputView.SetPlaceholder("Send a message or input a command").
		SetPlaceholderTextColor(tcell.ColorDeepSkyBlue)
	inputView.SetLabel(">").SetLabelColor(tcell.ColorDeepSkyBlue).SetLabelWidth(2)
	inputView.SetFieldTextColor(tcell.ColorWhite).SetFieldBackgroundColor(tcell.ColorGrey)

	inputSection := &InputSection{View: inputView, MessageBoard: messageBoard}
	inputView.SetDoneFunc(func(key tcell.Key) {
		text := inputView.GetText()
		if text == "" {
			return
		}
		messageBoard.HandleInput(strings.TrimSpace(text))
		inputView.SetText("")
	})
	return inputSection
}
