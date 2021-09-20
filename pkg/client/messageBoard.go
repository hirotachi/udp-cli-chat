package client

import (
	"encoding/json"
	"fmt"
	"github.com/hirotachi/udp-cli-chat/pkg/server"
	"github.com/rivo/tview"
)

type MessageBoard struct {
	View       *tview.TextView
	Frame      *tview.Frame
	Store      []*server.Message
	Connection *Connection
}

func NewMessageBoard(app *tview.Application, connection *Connection) *MessageBoard {
	messageView := tview.NewTextView().SetChangedFunc(func() {
		app.Draw()
	})
	messageView.SetDynamicColors(true).SetScrollable(true)

	//todo:add text to welcome user
	//todo:add text with commands and indication that if you want to see commands type /help in input
	messageFrame := tview.NewFrame(messageView)
	messageFrame.SetTitle("[#Cocus chat]").SetBorder(true).SetTitleAlign(0)
	messageBoard := &MessageBoard{
		View:       messageView,
		Frame:      messageFrame,
		Store:      make([]*server.Message, 0),
		Connection: connection,
	}
	go messageBoard.ListenToHistoryLoad()
	go messageBoard.ListenToMessages()
	go messageBoard.ListenToConnectionLog()

	return messageBoard
}

func (board *MessageBoard) ListenToHistoryLoad() {
	history := <-board.Connection.HistoryChan
	board.Store = history
	//todo stream history to view
}

func (board *MessageBoard) ListenToMessages() {
	for bytes := range board.Connection.MessageChan {
		var message server.Message
		if err := json.Unmarshal(bytes, &message); err != nil {
			board.Connection.LogError(fmt.Errorf("failed to unmarshal message: %s", err))
			return
		}
		board.Store = append(board.Store, &message)
		formattedMessage := GenerateMessageLog(&message)
		board.StreamToMessageView(formattedMessage...)
	}
}

func (board *MessageBoard) StreamToMessageView(data ...interface{}) {
	if _, err := fmt.Fprint(board.View, data...); err != nil {
		board.Connection.LogError(fmt.Errorf("failed to stream to message view: %s", err))
	}
}

// ListenToConnectionLog log errors and announcements to message board
func (board *MessageBoard) ListenToConnectionLog() {
	for log := range board.Connection.LogChan {
		board.StreamToMessageView("[red]error[::-]: ", log)
	}
}

func GenerateMessageLog(message *server.Message) []interface{} {
	date := message.CreatedAt.Format("Jan 2 15:04:05")
	date = fmt.Sprintf("[grey]%s[::-]", date)

	authorName := message.AuthorName
	if message.AuthorID != "" {
		authorName = fmt.Sprintf("[blue::b]%s[::-]", authorName)
	}
	return []interface{}{authorName, " ", date, "\n", "  [white]", message.Content, "\n\n"}
}
