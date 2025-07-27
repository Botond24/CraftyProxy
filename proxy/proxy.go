package proxy

import (
	"errors"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/data/packetid"
	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/server"
	"github.com/Tnze/go-mc/yggdrasil/user"
	"github.com/botond24/CraftyProxy/crafty"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"strconv"
)

var (
	messageOn  = "The server is starting, please try again in a minute."
	messageOff = "The server is stopped, please ask the owner to start it up"
)

func Handle(s *crafty.Server, addr string) {
	listen, err := net.Listen("tcp", addr+":"+strconv.Itoa(s.OutPort))
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
	if s.IsRunning() {
		forward(s, conn)
		return
	}
	startingReply(s, conn)
}

type LoginDenier struct {
	*crafty.Server
}

func (c *LoginDenier) AcceptLogin(conn *mcnet.Conn, protocol int32) (name string, id uuid.UUID, profilePubKey *user.PublicKey, properties []user.Property, err error) {
	if c.Server != nil {
		if c.AutoOn {
			c.Start()
			_ = conn.WritePacket(pk.Marshal(
				packetid.ClientboundLoginLoginDisconnect,
				chat.JsonMessage{Text: messageOn},
			))
			err = errors.New(messageOn)
		} else {
			_ = conn.WritePacket(pk.Marshal(
				packetid.ClientboundLoginLoginDisconnect,
				chat.JsonMessage{Text: messageOff},
			))
			err = errors.New(messageOff)
		}
	}
	return
}

type ServerInfo struct {
	*server.PlayerList
	*server.PingInfo
}

func (s ServerInfo) Protocol(clientProtocol int32) int {
	return int(clientProtocol)
}

func startingReply(s *crafty.Server, conn net.Conn) {
	playerList := server.NewPlayerList(1)
	pingInfo := server.NewPingInfo(s.Name, 0, chat.Text(messageOff), nil)
	if s.AutoOn {
		pingInfo = server.NewPingInfo(s.Name, 0, chat.Text("The server is stopped, you can start it by joining"), nil)
	}

	serverInfo := ServerInfo{
		PlayerList: playerList,
		PingInfo:   pingInfo,
	}

	srv := server.Server{
		Logger:          nil,
		ListPingHandler: serverInfo,
		LoginHandler: &LoginDenier{
			Server: s,
		},
		ConfigHandler: nil,
		GamePlay:      nil,
	}
	c := &mcnet.Conn{
		Socket: conn,
		Reader: conn,
		Writer: conn,
	}
	c.SetThreshold(-1)
	srv.AcceptConn(c)
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
		s.Logger.Println("User Connected")
		s.IncrementPlayers()
		defer s.DecrementPlayers()
		io.Copy(conn, serverConn)
	}()
	_, err = io.Copy(serverConn, conn)
	if err != nil {
		s.Logger.Println("Error copying data: " + err.Error())
	}
	return
}
