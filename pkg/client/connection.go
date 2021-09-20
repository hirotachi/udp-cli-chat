package client

import (
	"encoding/json"
	"fmt"
	"github.com/hirotachi/udp-cli-chat/pkg/server"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
	"net"
)

type Connection struct {
	AssignID           string
	conn               *net.UDPConn
	errorsLog          []error
	MessageChan        chan []byte
	LogChan            chan error
	HistoryChan        chan []*server.Message
	UDPMessagesQueue   chan []byte
	InitialHistory     []*server.Message
	LocalHistoryLength int
	isHistoryLoaded    bool
}

func NewConnection(address string, username string) (*Connection, error) {
	remoteAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %s", err)
	}
	conn, err := net.DialUDP("udp", nil, remoteAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to dial connection: %s", err)
	}

	connection := &Connection{
		conn:               conn,
		errorsLog:          make([]error, 0),
		MessageChan:        make(chan []byte),
		LogChan:            make(chan error),
		HistoryChan:        make(chan []*server.Message),
		LocalHistoryLength: 0,
		isHistoryLoaded:    false,
		InitialHistory:     make([]*server.Message, 0),
		UDPMessagesQueue:   make(chan []byte),
	}
	connection.RegisterClient(username)

	go connection.Listen()
	go connection.ListenToQueue()

	return connection, nil
}

func (c *Connection) Listen() {
	defer c.conn.Close()
	for {
		c.ReadUDPConnection()
	}
}

func (c *Connection) ReadUDPConnection() {
	bytes, _, err := utils.ReadUDPConn(c.conn)
	if err != nil {
		c.LogError(fmt.Errorf("failed to listen to UDP connection: %s", err))
		return
	}
	c.HandleUDPMessage(bytes)
}

func (c *Connection) HandleUDPMessage(msg []byte) {
	command, data := utils.ParseCommandAndData(msg)
	if !c.isHistoryLoaded { // while the history is not loaded completely add history messages
		switch command {
		case utils.InitialPayloadCommand:
			c.HandleInitialPayload(data)
		case utils.AddHistoryCommand:
			c.AddMessageToHistory(data)
		default:
			c.UDPMessagesQueue <- msg
		}
	} else {
		switch command {
		case utils.AddMessageCommand:
			c.MessageChan <- data
		case utils.DeleteMessageCommand:
		default:
			c.LogError(fmt.Errorf("unrecognized command from UDP connection: \"%s\"", command))
		}
	}

}

func (c *Connection) RegisterClient(username string) {
	loginInput := &server.LoginInput{
		Username: username,
	}
	if err := utils.WriteToUDPConn(c.conn, utils.ConnectCommand, loginInput); err != nil {
		c.LogError(fmt.Errorf("could not send connect command to UDP connection: %s", err))
	}
}

func (c *Connection) LogError(err error) {
	c.errorsLog = append(c.errorsLog, err)
	c.LogChan <- err
}

func (c *Connection) ListenToQueue() {
	for !c.isHistoryLoaded {
		//	don't do anything until history has been loaded completely
	}
	//	 run queued UDP messages
	for msg := range c.UDPMessagesQueue {
		c.HandleUDPMessage(msg)
	}
}

func (c *Connection) HandleInitialPayload(data []byte) {
	var initialPayload server.InitialPayload
	if err := json.Unmarshal(data, &initialPayload); err != nil {
		c.LogError(fmt.Errorf("failed to unmarshal initial payload"))
		return
	}
	if initialPayload.AssignedId == "" {
		c.LogError(fmt.Errorf("failed to receive assignedId"))
		return
	}

	c.AssignID = initialPayload.AssignedId
	if initialPayload.HistoryLength == 0 {
		c.isHistoryLoaded = true
		return
	}
	c.InitialHistory = make([]*server.Message, initialPayload.HistoryLength)
}

func (c *Connection) AddMessageToHistory(data []byte) {
	var historyLog server.HistoryLog
	if err := json.Unmarshal(data, &historyLog); err != nil {
		c.LogError(fmt.Errorf("could not unmarshal history log"))
		return
	}
	c.InitialHistory[historyLog.Order] = historyLog.Message
	c.LocalHistoryLength += 1

	c.isHistoryLoaded = len(c.InitialHistory) == c.LocalHistoryLength
	if c.isHistoryLoaded {
		c.HistoryChan <- c.InitialHistory
		close(c.UDPMessagesQueue)
		close(c.HistoryChan)
	}
}
