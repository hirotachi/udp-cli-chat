package client

import (
	"encoding/json"
	"fmt"
	"github.com/hirotachi/udp-cli-chat/pkg/server"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
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
	messageView.SetDynamicColors(true).SetScrollable(true).SetRegions(true)

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

	messageBoard.ShowWelcomeText()
	return messageBoard
}

func (board *MessageBoard) ListenToHistoryLoad() {
	history := <-board.Connection.HistoryChan
	board.Store = history
	historyLog := make([]interface{}, 0)
	for _, message := range history {
		msg := message
		formattedMessage := board.GenerateMessageLog(msg)
		historyLog = append(historyLog, formattedMessage...)
	}
	board.StreamToMessageView(historyLog...)
	board.View.ScrollToEnd()
}

func (board *MessageBoard) ListenToMessages() {
	for bytes := range board.Connection.MessageChan {
		var message server.Message
		if err := json.Unmarshal(bytes, &message); err != nil {
			board.Connection.LogError(fmt.Errorf("failed to unmarshal message: %s", err))
			return
		}
		board.Store = append(board.Store, &message)
		formattedMessage := board.GenerateMessageLog(&message)
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
		board.StreamToMessageView("[red]error[::-]: ", log, "\n\n")
	}
}

func (board *MessageBoard) HandleInput(text string) {
	switch text {
	case "/help":
		board.ListCommands()
	case "/disconnect":
		board.Connection.Disconnect()
	default:
		message := &server.Message{
			Content:  text,
			AuthorID: board.Connection.AssignID,
		}
		if err := utils.WriteToUDPConn(board.Connection.conn, utils.AddMessageCommand, message); err != nil {
			return
		}
	}
}

func (board *MessageBoard) ShowWelcomeText() {
	welcomeText := `[lightgrey::b]Welcome to Chat[::-]`
	board.StreamToMessageView(welcomeText, "\n\n")
	board.ListCommands()
}

type Option struct {
	Action      string
	Description string
	Prefix      string
}

func (board *MessageBoard) ListCommands() {
	commandsOptionsList := []Option{{
		Prefix:      "/",
		Action:      "help",
		Description: "Shows this commands list.",
	}, {
		Action:      "disconnect",
		Description: "Disconnects you and exists the program.",
		Prefix:      "/",
	}}

	arrowsOptionsList := []Option{
		{Prefix: "UP ARROW", Description: "When input is focused, message list is focused."},
		{Prefix: "ESC", Description: "Exit message list focus."},
	}

	arrows := BuildOptionsList("Keys", arrowsOptionsList)
	commands := BuildOptionsList("Commands", commandsOptionsList)
	if _, err := fmt.Fprint(board.View, commands, "\n", arrows); err != nil {
		board.Connection.LogError(fmt.Errorf("failed to list commands: %s", err))
	}
}

func BuildOptionsList(title string, optionsList []Option) string {
	result := fmt.Sprintf("[lightgrey::b]%s[::-] \n", title)
	for _, option := range optionsList {
		optionText := fmt.Sprintf("  [blue]%s[::-][white::b]%s[::-] [lightgrey]%s[::-]\n", option.Prefix, option.Action, option.Description)
		result += optionText
	}
	return result
}

func (board *MessageBoard) GenerateMessageLog(message *server.Message) []interface{} {
	date := message.CreatedAt.Format("Jan 2 15:04:05")
	info := fmt.Sprintf("[grey]%s[::-]", date)

	authorName := message.AuthorName
	if message.AuthorID == board.Connection.AssignID {
		authorName = fmt.Sprintf("[blue::b]%s[::-]", authorName)
	}
	return []interface{}{authorName, " ", info, "\n", "  [white]", message.Content, "[::-]\n\n"}
}
