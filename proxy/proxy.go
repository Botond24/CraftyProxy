package proxy

import (
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/Botond24/CraftyProxy/crafty"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/data/packetid"
	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/server"
	"github.com/Tnze/go-mc/yggdrasil/user"
	"github.com/google/uuid"
)

var (
	messageOn  = "The server is starting, please try again in a minute."
	messageOff = "The server is stopped, please ask the owner to start it up"
)

func Handle(s *crafty.Server, addr string) {
	listen, err := net.Listen("tcp", addr+":"+strconv.Itoa(int(s.OutPort)))
	if err != nil {
		log.Fatalf("Error starting proxy server: %s\n", err)
	}
	s.Logger.Println("TCP Proxy server started on port " + strconv.Itoa(int(s.OutPort)))
	s.Handled = true
	var udpListener *net.UDPConn = nil
	if s.VoiceOutPort != 0 {
		var udpAddr *net.UDPAddr = nil

		udpAddr, err = net.ResolveUDPAddr("udp", addr+":"+strconv.Itoa(s.VoiceOutPort))
		if err != nil {
			log.Fatalf("Error getting udpListener address: %s\n", err)
		}
		udpListener, err = net.ListenUDP("udp", udpAddr)
		if err != nil {
			log.Fatalf("Error starting proxy server: %s\n", err)
		}
		s.Logger.Println("UDP Proxy server started on port " + strconv.Itoa(s.VoiceOutPort))
	}
	for {
		if udpListener != nil && s.VoiceOutPort != 0 {
			go handleUDP(s, udpListener)
		}
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error accepting connection: " + err.Error())
			continue
		}
		go handleConnection(s, conn)
		if s.State == "removed" { // server was removed
			break
		}
	}
	if udpListener != nil {
		udpListener.Close()
	}
	listen.Close()
	s.Remove()
}

func handleUDP(s *crafty.Server, udp *net.UDPConn) {
	if s.IsRunning() {
		var buf [1500]byte
		n, addr, err := udp.ReadFromUDP(buf[0:])
		if err != nil {
			return
		}
		forwardUDP(s, udp, addr, n, buf)
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
	var p pk.Packet
	err = conn.ReadPacket(&p)
	if err != nil {
		return
	}
	err = p.Scan(
		(*pk.String)(&name), // decode username as pk.String
		(*pk.UUID)(&id),
	)
	if err != nil {
		return
	}
	if c.Server != nil {
		if c.AutoOn {
			c.Start(name)
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
	if s.State == "starting" {
		pingInfo = server.NewPingInfo(s.Name, 0, chat.Text("The server is starting, please wait"), nil)
	}
	serverInfo := ServerInfo{
		PlayerList: playerList,
		PingInfo:   pingInfo,
	}

	srv := server.Server{
		Logger:          s.Logger,
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
	serverConn, err := net.Dial("tcp", s.Address+":"+strconv.Itoa(int(s.InPort)))
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

var mu sync.Mutex
var conns = make(map[string]*net.UDPConn)

func forwardUDP(s *crafty.Server, udp *net.UDPConn, clientAddr *net.UDPAddr, n int, b [1500]byte) {
	port := s.VoiceInPort
	addr, err := net.ResolveUDPAddr("udp", s.Address+":"+strconv.Itoa(port))
	if err != nil {
		s.Logger.Fatalf("Error getting udp address: %s\n", err)
	}
	mu.Lock()
	conn, exists := conns[clientAddr.String()]
	if !exists {
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			s.Logger.Fatalf("Error connecting to udp address%s: %s\n", clientAddr, err)
		}
		conns[clientAddr.String()] = conn
	}
	mu.Unlock()
	_, err = conn.Write(b[:n])
	if err != nil {
		s.Logger.Fatalf("Error writing to udp address %s: %s\n", clientAddr, err)
	}

	s.Logger.Println("Voice connected")
	go func() {
		buf := make([]byte, 1500)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				s.Logger.Printf("Server read error (%s): %v\n", clientAddr, err)
				return
			}
			_, err = udp.WriteToUDP(buf[:n], clientAddr)
			if err != nil {
				s.Logger.Printf("Client write error (%s): %v\n", clientAddr, err)
				return
			}
		}
	}()
}
