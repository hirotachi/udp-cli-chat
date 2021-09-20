package main

import "github.com/hirotachi/udp-cli-chat/pkg/client"

func main() {
	udpClient, err := client.NewUDPClient()
	if err != nil {
		panic(err)
	}
	if err := udpClient.Run(); err != nil {
		panic(err)
	}
}
