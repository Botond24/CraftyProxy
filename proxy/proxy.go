package proxy

import (
	"CraftyProxy/crafty"
	"io"
	"log"
	"net"
	"strconv"
)

func Handle(s *crafty.Server) {
	listen, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(s.OutPort))
	if err != nil {
		log.Fatalf("Error starting proxy server: %s\n", err)
	}
	s.Logger.Println("Proxy server started on port " + strconv.Itoa(s.OutPort))
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error accepting connection: " + err.Error())
			continue
		}
		go handleConnection(s, conn)
	}
}

func handleConnection(s *crafty.Server, conn net.Conn) {
	s.IncrementPlayers()
	defer s.DecrementPlayers()
	if s.IsRunning() {
		forward(s, conn)
		return
	}
	s.Start()
	forward(s, conn)
}

func forward(s *crafty.Server, conn net.Conn) {
	serverConn, err := net.Dial("tcp", s.Address+":"+strconv.Itoa(s.InPort))
	if err != nil {
		s.Logger.Println("Error connecting to server: " + err.Error())
		return
	}
	s.Logger.Println("Connected to server")
	defer serverConn.Close()
	defer conn.Close()
	go func() {
		_, err = io.Copy(conn, serverConn)
		s.Logger.Println("User Connected")
		if err != nil {
			s.Logger.Println("Error copying data: " + err.Error())
		}
	}()
	_, err = io.Copy(serverConn, conn)
	if err != nil {
		s.Logger.Println("Error copying data: " + err.Error())
	}
	return
}
