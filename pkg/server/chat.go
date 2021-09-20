package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
	"github.com/rs/xid"
	"log"
	"net"
	"time"
)

type Chat struct {
	RedisClient   *redis.Client
	conn          *net.UDPConn
	History       []*Message
	Clients       map[string]*Client
	BroadcastChan chan []byte
	MessageChan   chan Message
	connected     int
	HistoryLimit  int
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
		MessageChan:   make(chan Message),
		connected:     0,
		HistoryLimit:  20,
	}
}

func (chat *Chat) Listen() {
	defer chat.conn.Close()
	go chat.ListenToChannels()
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
	case utils.AddMessageCommand:
		chat.AddMessage(data, addr)
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

func (chat *Chat) ListenToChannels() {
	// iterate over all clients
	forEachClient := func(isOnline bool, handler func(client *Client)) {
		for _, client := range chat.Clients {
			c := client
			if isOnline == c.Online {
				handler(c)
			}
		}
	}
	for {
		select {
		case msg := <-chat.BroadcastChan:
			forEachClient(true, func(client *Client) {
				client.BroadcastChan <- msg
			})
		case msg := <-chat.MessageChan:
			forEachClient(true, func(client *Client) {
				if client.ID != msg.AuthorID { // hide other clients ids from client
					msg.AuthorID = ""
				}
				client.MessageChan <- &msg
			})
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

	// send each history log by itself to avoid data loss
	for i, message := range chat.History {
		m := *message // copy to avoid mutating message in history
		authorName := "guest"
		author, ok := chat.Clients[m.AuthorID]
		if ok {
			authorName = author.Name
		}
		m.AuthorName = authorName // author name to message to be identified by other clients
		if m.AuthorID != client.ID {
			m.AuthorID = ""
		}
		historyLog := &HistoryLog{
			Order:   i,
			Message: &m,
		}
		utils.BroadcastWithCommand(client.BroadcastChan, utils.AddHistoryCommand, historyLog)
	}
}

func (chat *Chat) AddMessage(data []byte, addr *net.UDPAddr) {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		log.Println("failed to unmarshal message: ", err)
		return
	}
	var client *Client
	if message.AuthorID != "" { // check if client exists before saving message
		var ok bool
		client, ok = chat.Clients[message.AuthorID]
		if !ok {
			log.Printf("Unrecognized client \"%s\" with id \"%s\"\n", addr, message.AuthorID)
			return
		}
	}
	message.ID = xid.New().String()
	message.CreatedAt = time.Now()
	if err := chat.SaveMessageToRedis(&message); err != nil {
		log.Println(err)
		return
	}
	history := chat.History
	if len(history) == chat.HistoryLimit { // limit history
		history = history[len(history)-19:]
	}
	chat.History = append(chat.History, &message)
	message.AuthorName = client.Name // add author name to be recognized by other clients

	chat.MessageChan <- message
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

func (chat *Chat) SaveMessageToRedis(message *Message) error {
	ctx := context.Background()
	historyLength, err := chat.RedisClient.LLen(ctx, utils.RedisHistoryKey).Result()
	if err != nil {
		log.Println("failed to fetch history length from redis: ", err)
	}

	if historyLength == int64(historyLength) { // limit history log on redis
		if err := chat.RedisClient.LTrim(ctx, utils.RedisHistoryKey, 1, -1).Err(); err != nil {
			return fmt.Errorf("failed to limit redis history to 20 entries: %s", err)
		}
	}

	bytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %s", err)
	}
	if err := chat.RedisClient.RPush(ctx, utils.RedisHistoryKey, string(bytes)).Err(); err != nil {
		return fmt.Errorf("failed to save message to redis history: %s", err)
	}
	return nil
}
