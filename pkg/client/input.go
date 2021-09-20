package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

type InputSection struct {
	View         *tview.InputField
	MessageBoard *MessageBoard
	Focus        func(view string)
}

func NewInputSection(messageBoard *MessageBoard) *InputSection {
	inputView := tview.NewInputField()
	inputView.SetPlaceholder("Send a message or input a command ot /help to list commands").
		SetPlaceholderTextColor(tcell.ColorDeepSkyBlue)
	inputView.SetLabel(">").SetLabelColor(tcell.ColorDeepSkyBlue).SetLabelWidth(2)
	inputView.SetFieldTextColor(tcell.ColorWhite).SetFieldBackgroundColor(tcell.ColorGrey)

	inputSection := &InputSection{View: inputView, MessageBoard: messageBoard}
	inputView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			text := inputView.GetText()
			if text == "" {
				return
			}
			messageBoard.HandleInput(strings.TrimSpace(text))
			inputView.SetText("")
		case tcell.KeyUp:
			inputSection.Focus(MessageView)
		}
	})
	return inputSection
}
