package server

import (
	"github.com/rs/xid"
	"log"
	"net"
)

type Client struct {
	Name          string        `json:"name"`
	Address       *net.UDPAddr  `json:"-"`
	Online        bool          `json:"online"`
	ID            string        `json:"id,omitempty"`
	conn          *net.UDPConn  `json:"-"`
	BroadcastChan chan []byte   `json:"-"`
	MessageChan   chan *Message `json:"-"`
}

func NewClient(chat *Chat, addr *net.UDPAddr, username string) *Client {
	return &Client{
		Name:          username,
		Address:       addr,
		Online:        true,
		ID:            xid.New().String(),
		conn:          chat.conn,
		BroadcastChan: make(chan []byte),
		MessageChan:   make(chan *Message),
	}
}

func (c *Client) Listen() {
	for {
		select {
		case msg := <-c.BroadcastChan:
			c.SendMessage(msg)
		}
	}
}

func (c *Client) SendMessage(msg []byte) {
	_, err := c.conn.WriteToUDP(msg, c.Address)
	if err != nil {
		log.Printf("failed to send message to %s: %s\n", c.Address, err)
	}
}
