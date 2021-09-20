package server

import (
	"context"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/hirotachi/udp-cli-chat/pkg/utils"
	"log"
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
