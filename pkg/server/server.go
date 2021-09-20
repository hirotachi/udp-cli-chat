package server

import (
	"github.com/go-redis/redis/v8"
	"net"
)

type Server struct {
	UDPAddr     *net.UDPAddr
	conn        *net.UDPConn
	RedisClient *redis.Client
}

func (s *Server) Run() error {
	var err error
	s.conn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	chat := NewChat(s)

	chat.Listen()
	return nil
}

func NewServer(address string, redisClient *redis.Client) (*Server, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return nil, err
	}
	server := &Server{
		UDPAddr:     udpAddr,
		RedisClient: redisClient,
	}
	return server, nil
}
