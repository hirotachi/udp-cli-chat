package server

import (
	"context"
	"encoding/json"
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

	clientLoginInput := &LoginInput{
		Username: "tester",
	}
	var initialPayload InitialPayload

	t.Run("Sending first connect with username returns assignedID and history length", func(t *testing.T) {
		if err := utils.WriteToUDPConn(conn, utils.ConnectCommand, clientLoginInput); err != nil {
			t.Error("could not write to UDP connection: ", err)
		}
		bytes, _, err := utils.ReadUDPConn(conn)
		if err != nil {
			t.Error("could not read from UDP connection: ", err)
		}
		command, data := utils.ParseCommandAndData(bytes)
		assert.Equal(t, utils.InitialPayloadCommand, command)
		UnpackTestData(t, data, &initialPayload)
		assert.NotEmpty(t, initialPayload.AssignedId)
		assert.Equal(t, initialPayload.HistoryLength, 0)

		clientLength, err := server.RedisClient.SCard(ctx, utils.RedisClientsSetKey).Result()
		if err != nil {
			t.Error("failed to fetch clients set length from redis")
		}
		assert.Equal(t, int64(1), clientLength)
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
