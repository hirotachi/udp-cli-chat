package main

import "github.com/hirotachi/udp-cli-chat/pkg/client"

func main() {
	udpClient := client.NewUDPClient()
	if err := udpClient.Run(); err != nil {
		panic(err)
	}
}
