package main

import (
	"context"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/hirotachi/udp-cli-chat/pkg/server"
	"log"
)

func main() {
	// temporary redis server for development
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalln("error creating redis db: ", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalln("cannot connect to redis db: ", err)
	}

	serverAddress := ":5000"
	udpServer, err := server.NewServer(serverAddress, redisClient)
	if err != nil {
		log.Fatalln("error creating UDP server: ", err)
	}
	if err := udpServer.Run(); err != nil {
		panic(err)
	}
}
