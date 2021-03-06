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
	clientsMap, connected := FetchClientsFromRedis(server.RedisClient)
	history := FetchHistoryFromRedis(server.RedisClient)
	return &Chat{
		RedisClient:   server.RedisClient,
		conn:          server.conn,
		History:       history,
		Clients:       clientsMap,
		BroadcastChan: make(chan []byte),
		MessageChan:   make(chan Message),
		connected:     connected,
		HistoryLimit:  20,
	}
}

func FetchHistoryFromRedis(redisClient *redis.Client) []*Message {
	history := make([]*Message, 0)
	if err := redisClient.LRange(context.Background(), utils.RedisHistoryKey, 0, -1).ScanSlice(&history); err != nil && err != redis.Nil {
		log.Println("could not fetch redis messages history: ", err)
	}
	return history
}

func FetchClientsFromRedis(redisClient *redis.Client) (map[string]*Client, int) {
	clients := make([]*Client, 0)
	if err := redisClient.SMembers(context.Background(), utils.RedisClientsSetKey).ScanSlice(&clients); err != nil && err != redis.Nil {
		log.Println("could not fetch redis clients list: ", err)
	}
	clientsByIdMap := map[string]*Client{}
	connected := 0
	if len(clients) != 0 {
		for _, c := range clients {
			client := c
			clientsByIdMap[client.ID] = client
			if client.Online {
				connected++
			}
		}
	}
	return clientsByIdMap, connected
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
	go func() {
		switch command {
		case utils.ConnectCommand:
			chat.Join(addr, data)
		case utils.AddMessageCommand:
			chat.AddMessage(data, addr)
		case utils.DeleteMessageCommand:
			chat.DeleteMessage(data, addr)
		case utils.DisconnectCommand:
			chat.Disconnect(data, addr)
		default:
			log.Printf("unknown command \"%s\" from address: %s\n", command, addr)
		}
	}()
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

func (chat *Chat) Disconnect(data []byte, addr *net.UDPAddr) {
	clientID := string(data)
	client, ok := chat.Clients[clientID]
	if !ok {
		log.Printf("client \"%s\" does exist\n", clientID)
		return
	}
	clientBytes, err := json.Marshal(client)
	if err != nil {
		log.Printf("could not marshal client \"%s\" for redis removal: %s\n", clientID, err)
		return
	}

	// remove client from redis to re-add with updated status
	if err := chat.RedisClient.SRem(context.Background(), utils.RedisClientsSetKey, string(clientBytes)).Err(); err != nil {
		log.Printf("could not remove client  \"%s\" from redis set: %s\n", clientID, err)
		return
	}
	client.Online = false
	if err := chat.SaveClientToRedis(client); err != nil {
		log.Println(err)
		return
	}
	chat.connected -= 1
	if chat.connected == 0 { // clear messages history
		if err := chat.RedisClient.Del(context.Background(), utils.RedisHistoryKey).Err(); err != nil {
			log.Println("failed to empty redis history: ", err)
			return
		}
		chat.History = make([]*Message, 0)
	}
	log.Printf("client \"%s\" disconnected\n", addr)
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
				message := msg
				if client.ID != message.AuthorID { // hide other clients ids from client
					message.AuthorID = ""
				}
				client.MessageChan <- &message
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
	msg := message // copy so message doesn't get mutated
	chat.History = append(chat.History, &msg)
	message.AuthorName = client.Name // add author name to be recognized by other clients

	chat.MessageChan <- message
}

func (chat *Chat) DeleteMessage(data []byte, addr *net.UDPAddr) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Println("failed to unmarshal deleted message: ", err)
		return
	}
	if msg.AuthorID == "" || msg.ID == "" {
		log.Printf("failed to delete message from \"%s\"\n", addr)
		return
	}
	_, ok := chat.Clients[msg.AuthorID]
	if !ok {
		log.Printf("failed to delete message from \"%s\"\n", addr)
		return
	}

	msg.AuthorName = "" // remove author_name to find on redis list
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("failed to marshal msg for redis deletion: ", err)
		return
	}
	removedCount, err := chat.RedisClient.LRem(context.Background(), utils.RedisHistoryKey, 1, string(msgBytes)).Result()
	if err != nil {
		log.Println("could not remove msg from redis: ", err)
		return
	}
	if removedCount == 0 {
		log.Printf("message \"%s\" doesnt exists on redis to be removed\n", msg.ID)
		return
	}

	newHistory := make([]*Message, 0)
	for _, message := range chat.History {
		m := message
		if m.ID != msg.ID {
			newHistory = append(newHistory, m)
		}
	}
	chat.History = newHistory
	utils.BroadcastWithCommand(chat.BroadcastChan, utils.DeleteMessageCommand, msg.ID)
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

	if historyLength == int64(chat.HistoryLimit) { // limit history log on redis
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
