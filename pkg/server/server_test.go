package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"testing"
)

var server *Server

const serverAddress = ":1123"

func init() {
	mr, err := miniredis.Run()
	if err != nil {
		log.Println("error creating redis db: ", err)
		return
	}
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if _, err := redisClient.Ping(context.TODO()).Result(); err != nil {
		log.Println("cannot connect to redis db: ", err)
		return
	}
	server, err = NewServer(serverAddress, redisClient)
	if err != nil {
		log.Println("error creating UDP server")
		return
	}
	go func() {
		server.Run()
	}()
}

func TestNetServer_Run(t *testing.T) {
	conn, err := utils.GetUDPConnection(serverAddress)
	if err != nil {
		t.Error("could not create UDP connection: ", err)
	}
	defer conn.Close()
}

func TestNetServer_Request(t *testing.T) {
	ctx := context.TODO()

	conn := CreateTestConnection(t, serverAddress)
	defer conn.Close()

	secondConn := CreateTestConnection(t, serverAddress)
	defer secondConn.Close()

	clientLoginInput := &LoginInput{
		Username: "tester",
	}
	secondClientLoginInput := &LoginInput{
		Username: "tester2",
	}

	var initialPayload *InitialPayload
	var receivedMessage Message

	t.Run("Sending first connect with username returns assignedID and history length", func(t *testing.T) {
		initialPayload = AddTestClient(t, conn, clientLoginInput)
		assert.NotEmpty(t, initialPayload.AssignedId)
		assert.Equal(t, 0, initialPayload.HistoryLength)

		clientLength, err := server.RedisClient.SCard(ctx, utils.RedisClientsSetKey).Result()
		if err != nil {
			t.Error("failed to fetch clients set length from redis")
		}
		assert.Equal(t, int64(1), clientLength)
	})

	t.Run("Sending add message request with message broadcasts message to all clients", func(t *testing.T) {
		message := &Message{
			Content:  "hello world",
			AuthorID: initialPayload.AssignedId,
		}
		if err := utils.WriteToUDPConn(conn, utils.AddMessageCommand, message); err != nil {
			t.Error("could not write to UDP connection: ", err)
		}
		bytes, _, err := utils.ReadUDPConn(conn)
		if err != nil {
			t.Error("could not read from UDP connection: ", err)
		}
		command, data := utils.ParseCommandAndData(bytes)
		assert.Equal(t, utils.AddMessageCommand, command)
		UnpackTestData(t, data, &receivedMessage)
		assert.NotEmpty(t, receivedMessage.ID)
		assert.Equal(t, initialPayload.AssignedId, receivedMessage.AuthorID)
		assert.Equal(t, clientLoginInput.Username, receivedMessage.AuthorName)
		assert.Equal(t, message.Content, receivedMessage.Content)
		assert.Empty(t, receivedMessage.Edited)
		assert.NotEmpty(t, receivedMessage.CreatedAt)

		historyLength, err := server.RedisClient.LLen(ctx, utils.RedisHistoryKey).Result()
		if err != nil {
			t.Error("failed to get history length from redis: ", err)
		}
		assert.Equal(t, int64(1), historyLength)
	})

	var secondConHistory int
	t.Run("Sending another connect with new client receives  initialPayload with history length", func(t *testing.T) {
		secondConInitialPayload := AddTestClient(t, secondConn, secondClientLoginInput)
		assert.NotEmpty(t, secondConInitialPayload.AssignedId)
		assert.Equal(t, 1, secondConInitialPayload.HistoryLength)
		secondConHistory = secondConInitialPayload.HistoryLength

		clientLength, err := server.RedisClient.SCard(ctx, utils.RedisClientsSetKey).Result()
		if err != nil {
			t.Error("failed to fetch clients set length from redis")
		}
		assert.Equal(t, int64(2), clientLength)
	})

	t.Run(fmt.Sprintf("Adding a new client returns %d history logs with order", secondConHistory), func(t *testing.T) {
		bytes, _, err := utils.ReadUDPConn(secondConn)
		if err != nil {
			t.Error("could not read from second UDP connection: ", err)
		}
		command, data := utils.ParseCommandAndData(bytes)
		assert.Equal(t, utils.AddHistoryCommand, command)
		var historyLog HistoryLog
		UnpackTestData(t, data, &historyLog)
		assert.Equal(t, 0, historyLog.Order)
	})

	t.Run("Sending delete message request broadcasts message deletion to all clients", func(t *testing.T) {
		if err := utils.WriteToUDPConn(conn, utils.DeleteMessageCommand, receivedMessage); err != nil {
			t.Error("could not write to UDP connection: ", err)
		}
		bytes, _, err := utils.ReadUDPConn(conn)
		if err != nil {
			t.Error("could not read from second UDP connection: ", err)
		}
		command, data := utils.ParseCommandAndData(bytes)
		assert.Equal(t, utils.DeleteMessageCommand, command)
		assert.Equal(t, receivedMessage.ID, string(data))

		historyLength, err := server.RedisClient.LLen(ctx, utils.RedisHistoryKey).Result()
		if err != nil {
			t.Error("failed to get history length from redis: ", err)
		}
		assert.Equal(t, int64(0), historyLength)
	})
}

func CreateTestConnection(t *testing.T, address string) *net.UDPConn {
	conn, err := utils.GetUDPConnection(address)
	if err != nil {
		t.Error("could not connect to server: ", err)
	}
	return conn
}

func UnpackTestData(t *testing.T, data []byte, target interface{}) {
	if err := json.Unmarshal(data, target); err != nil {
		t.Error("could not unmarshal messages list")
	}
}

func AddTestClient(t *testing.T, conn *net.UDPConn, loginInput *LoginInput) *InitialPayload {
	if err := utils.WriteToUDPConn(conn, utils.ConnectCommand, loginInput); err != nil {
		t.Error("could not write to UDP connection: ", err)
	}
	bytes, _, err := utils.ReadUDPConn(conn)
	if err != nil {
		t.Error("could not read from UDP connection: ", err)
	}
	command, data := utils.ParseCommandAndData(bytes)
	assert.Equal(t, utils.InitialPayloadCommand, command)
	var initialPayload InitialPayload
	UnpackTestData(t, data, &initialPayload)
	return &initialPayload
}
