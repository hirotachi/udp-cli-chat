package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
)

// GetUDPConnection create udp connection from address string.
func GetUDPConnection(address string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, udpAddr)
}

// WriteToUDPConn marshals data and combines it with command and sends it to connection.
func WriteToUDPConn(conn *net.UDPConn, command string, data interface{}) error {
	var bytes []byte
	var err error
	switch command {
	case DisconnectCommand:
		bytes = []byte(data.(string))
	default:
		bytes, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("could not marshal data: %s", err)
		}
	}
	_, err = conn.Write(append([]byte(command), bytes...))
	return err
}

// ReadUDPConn read from UDP connection
func ReadUDPConn(conn *net.UDPConn) ([]byte, *net.UDPAddr, error) {
	out := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(out)
	if err != nil {
		return nil, nil, err
	}
	return out[:n], addr, nil
}

// ParseCommandAndData reads received bytes and split commands and data
func ParseCommandAndData(msg []byte) (string, []byte) {
	str := string(msg)
	split := strings.SplitAfter(str, ">")
	command := strings.TrimSpace(split[0])
	data := split[1]
	return command, []byte(data)
}

// BroadcastWithCommand sends marshaled data to passed in channel
func BroadcastWithCommand(channel chan []byte, command string, data interface{}) {
	msg := BuildUDPMessage(command, data)
	if msg == nil {
		return
	}
	channel <- msg
}

func BuildUDPMessage(command string, data interface{}) []byte {
	var bytes []byte
	var err error
	switch command {
	case DeleteMessageCommand:
		bytes = []byte(data.(string))
	default:
		bytes, err = json.Marshal(data)
		if err != nil {
			log.Printf("failed to marshal command %s data: %s\n", command, err)
			return nil
		}
	}

	return append([]byte(command), bytes...)
}
