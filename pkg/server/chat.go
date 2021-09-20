package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
	"log"
	"net"
)

type Chat struct {
	RedisClient   *redis.Client
	conn          *net.UDPConn
	History       []*Message
	Clients       map[string]*Client
	BroadcastChan chan []byte
	MessageChan   chan *Message
	connected     int
}

type InitialPayload struct {
	AssignedId    string `json:"assigned_id,omitempty"`
	HistoryLength int    `json:"history_length,omitempty"`
}

func NewChat(server *Server) *Chat {
	//	 todo fetch messages from redis
	//	 todo fetch clients from redis
	return &Chat{
		RedisClient:   server.RedisClient,
		conn:          server.conn,
		History:       make([]*Message, 0),
		Clients:       map[string]*Client{},
		BroadcastChan: make(chan []byte),
		MessageChan:   make(chan *Message),
		connected:     0,
	}
}

func (chat *Chat) Listen() {
	defer chat.conn.Close()
	for {
		chat.HandleUDPConnection()
	}
}

func (chat *Chat) HandleUDPConnection() {
	bytes, addr, err := utils.ReadUDPConn(chat.conn)
	if err != nil {
		log.Printf("cannot read from %s connection: %s\n", addr, err)
		return
	}
	command, data := utils.ParseCommandAndData(bytes)
	switch command {
	case utils.ConnectCommand:
		chat.Join(addr, data)
	default:
		log.Printf("unknown command \"%s\" from address: %s\n", command, addr)
	}
}

func (chat *Chat) Join(addr *net.UDPAddr, data []byte) {
	ctx := context.Background()

	var client *Client
	var oldClient Client // to be deleted from redis if client is reconnecting

	username := "guest"
	var loginInput LoginInput
	if err := json.Unmarshal(data, &loginInput); err != nil {
		log.Println("failed to unmarshal login input")
	}
	if loginInput.Username != "" {
		username = loginInput.Username
	}

	if loginInput.AssignedId != "" {
		c, ok := chat.Clients[loginInput.AssignedId]
		if ok {
			client = c
			oldClient = *c
		}
		if client.Name != loginInput.Username && loginInput.Username != "" { // in case user decided to change when reconnecting
			client.Name = loginInput.Username
		}
		client.Address = addr
		client.Online = true
	}

	if client == nil {
		client = NewClient(chat, addr, username)
	}

	if (oldClient != Client{}) {
		//	todo remove old client from redis
		bytes, err := json.Marshal(oldClient)
		if err != nil {
			log.Println("could not marshal old client to be saved to redis: ", err)
			return
		}
		if err := chat.RedisClient.SRem(ctx, utils.RedisClientsSetKey, string(bytes)).Err(); err != nil {
			log.Println("failed to remove old client from redis set: ", err)
			return
		}
	}
	if err := chat.SaveClientToRedis(client); err != nil {
		log.Println(err)
		return
	}
	chat.Clients[client.ID] = client
	if !oldClient.Online {
		chat.connected += 1
	}

	go client.Listen()
	log.Printf("client \"%s\" connected\n", addr)

	go chat.SendInitialPayload(client)
}

func (chat *Chat) ListenToBroadCast() {
	for msg := range chat.BroadcastChan {
		for _, client := range chat.Clients {
			c := client
			if c.Online {
				c.BroadcastChan <- msg
			}
		}
	}
}

func (chat *Chat) SendInitialPayload(client *Client) {
	// send info to client to receive history logs split packets
	initialPayload := &InitialPayload{
		AssignedId:    client.ID,
		HistoryLength: len(chat.History),
	}
	utils.BroadcastWithCommand(client.BroadcastChan, utils.InitialPayloadCommand, initialPayload)

}

func (chat *Chat) SaveClientToRedis(client *Client) error {
	bytes, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("could not marshal client to be saved to redis: %s", err)
	}
	if err := chat.RedisClient.SAdd(context.Background(), utils.RedisClientsSetKey, string(bytes)).Err(); err != nil {
		return fmt.Errorf("could not save client to redis set: %s", err)
	}
	return nil
}
