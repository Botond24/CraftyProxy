package crafty

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type fileResponse struct {
	Data string `json:"data"`
}

type Server struct {
	Name      string
	parent    *Crafty
	InPort    int
	OutPort   int
	AutoOn    bool
	autoOff   bool
	id        string
	Logger    *log.Logger
	Address   string
	players   int
	stopTimer *time.Timer
	State     string
}

func NewServer(parent *Crafty, srv jsonServer) *Server {
	s := new(Server)
	s.AutoOn = false
	s.autoOff = false
	s.parent = parent
	s.InPort = srv.Port - 2000
	s.OutPort = srv.Port
	s.id = srv.Id
	s.players = 0
	s.Address = srv.Ip
	if srv.Ip == "127.0.0.1" {
		s.Address = parent.ip
	}
	options := strings.Split(srv.Name, "$")
	if len(options) > 1 {
		s.Name = strings.TrimSpace(options[0])
		options = strings.Split(options[1], "&")
	}
	s.Logger = log.New(os.Stdout, "crafty("+parent.ip+") | "+s.Name+": ", 0)

	if slices.Contains(options, "player-start") {
		s.AutoOn = true
	}
	if slices.Contains(options, "player-stop") {
		s.autoOff = true
	}
	if s.AutoOn || s.autoOff {
		s.updatePort()
	}
	s.stopTimer = time.AfterFunc(parent.StopTimeout*time.Minute, func() {
		s.Stop()
	})
	s.stopTimer.Stop()
	s.State = "unknown"
	return s
}

func (s *Server) String() string {
	return s.Name + " (" + strconv.Itoa(s.OutPort) + "->" + strconv.Itoa(s.InPort) + ")" + "\n" +
		"\tAuto on: " + strconv.FormatBool(s.AutoOn) + "\n" +
		"\tAuto off: " + strconv.FormatBool(s.autoOff) + "\n" +
		"\tID: " + s.id
}

func (s *Server) Start() {
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/action/start_server", []byte{})
	if err != nil {
		s.Logger.Println("Can't start server: " + err.Error())
	}
	if post.StatusCode != 200 {
		s.Logger.Println("Can't start server: " + post.Status)
		return
	}
	s.Logger.Println("Started server")
	s.State = "starting"
}

func (s *Server) Stop() {
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/action/stop_server", []byte{})
	if err != nil {
		s.Logger.Println("Can't stop server: " + err.Error())
	}
	if post.StatusCode != 200 {
		s.Logger.Println("Can't stop server: " + post.Status)
		return
	}
	s.Logger.Println("Stopped server")
	s.State = "stopping"
}

func (s *Server) updatePort() {
	path := "servers/" + s.id + "/server.properties"
	props := "{\"path\":\"" + path + "\"}"
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/files", []byte(props))
	if err != nil {
		s.Logger.Println("Can't update port: " + err.Error())
	}
	body := defaultServerProperties
	if post.StatusCode != 200 {
		s.Logger.Println("server.properties not found, creating...")
		s.createProperties()
	} else {
		defer post.Body.Close()
		postBody, err := io.ReadAll(post.Body)
		if err != nil {
			s.Logger.Println("Can't read response body: " + err.Error())
		}
		var file fileResponse
		json.Unmarshal(postBody, &file)
		body = file.Data
	}

	body = strings.ReplaceAll(body, strconv.Itoa(s.OutPort), strconv.Itoa(s.InPort))
	body = strings.ReplaceAll(body, "25565", strconv.Itoa(s.InPort))
	body = "{\"path\":\"" + path + "\",\"contents\":\"" + strings.ReplaceAll(
		strings.ReplaceAll(body, "\n", "\\n"), "\\:", "\\\\:") + "\"}"
	patch, err := s.parent.Patch("/api/v2/servers/"+s.id+"/files", []byte(body))
	if err != nil {
		s.Logger.Println("Can't update server.properties: " + err.Error())
	}
	if patch.StatusCode != 200 {
		s.Logger.Println("Can't update server.properties: " + patch.Status)
		defer patch.Body.Close()
		patchBody, err := io.ReadAll(patch.Body)
		if err != nil {
			s.Logger.Println("Can't read response body: " + err.Error())
		}
		s.Logger.Println(string(patchBody))
	}

}

func (s *Server) createProperties() {
	body := "{\"parent\":\"servers/" + s.id + "\",\"name\": \"server.property\",\"directory\": false}"
	put, err := s.parent.Put("/api/v2/servers/"+s.id+"/files/servers/"+s.id+"/server.properties", []byte(body))
	if err != nil {
		s.Logger.Println("Can't create server.properties: " + err.Error() + "\n")
	}
	if put.StatusCode != 200 {
		s.Logger.Println("Can't create server.properties: " + put.Status + "\n")
	}
}

func (s *Server) IsRunning() bool {
	get, err := s.parent.Get("/api/v2/servers/" + s.id + "/stats")
	if err != nil {
		s.Logger.Println("Can't get server stats: " + err.Error() + "\n")
		return false
	}
	defer get.Body.Close()
	body, err := io.ReadAll(get.Body)
	if err != nil {
		s.Logger.Println("Can't read response body: " + err.Error() + "\n")
		return false
	}
	var stats map[string]interface{}
	err = json.Unmarshal(body, &stats)
	if err != nil {
		s.Logger.Println("Can't extract JSON to object: " + err.Error() + "\n")
		return false
	}
	isrunning := stats["data"].(map[string]interface{})["running"].(bool)
	if isrunning {
		s.State = "running"
	} else {
		s.State = "stopped"
	}
	return isrunning
}

func (s *Server) IncrementPlayers() {
	s.players++
	s.Logger.Println("Players: " + strconv.Itoa(s.players))
	if s.autoOff {
		s.stopTimer.Stop()
	}
}

func (s *Server) DecrementPlayers() {
	s.players--
	s.Logger.Println("Players: " + strconv.Itoa(s.players))
	if s.players <= 0 {
		s.players = 0
		if s.autoOff {
			s.Logger.Println("Stopping server in " + strconv.Itoa(int(s.parent.StopTimeout)) + " minutes")
			s.stopTimer.Reset(s.parent.StopTimeout * time.Minute)
		}

	}
}

const (
	defaultServerProperties = "allow-flight=true\\n" +
		"allow-nether=true\\n" +
		"broadcast-console-to-ops=true\\n" +
		"broadcast-rcon-to-ops=true\\n" +
		"difficulty=normal\\n" +
		"enable-command-block=false\\n" +
		"enable-jmx-monitoring=false\\n" +
		"enable-query=false\\n" +
		"enable-rcon=false\\n" +
		"enable-status=true\\n" +
		"enforce-secure-profile=true\\n" +
		"enforce-whitelist=false\\n" +
		"entity-broadcast-range-percentage=100\\n" +
		"force-gamemode=false\\n" +
		"function-permission-level=2\\n" +
		"gamemode=survival\\n" +
		"generate-structures=true\\n" +
		"generator-settings={}\\n" +
		"hardcore=false\\n" +
		"hide-online-players=false\\n" +
		"initial-disabled-packs=\\n" +
		"initial-enabled-packs=vanilla\\n" +
		"level-name=world\\n" +
		"level-seed=\\n" +
		"level-type=minecraft\\:normal\\n" +
		"max-chained-neighbor-updates=1000000\\n" +
		"max-players=20\\n" +
		"max-tick-time=60000\\n" +
		"max-world-size=29999984\\n" +
		"motd=A Fanatastic server\\n" +
		"network-compression-threshold=256\\n" +
		"online-mode=true\\n" +
		"op-permission-level=4\\n" +
		"player-idle-timeout=0\\n" +
		"prevent-proxy-connections=false\\n" +
		"pvp=true\\n" +
		"query.port=25565\\n" +
		"rate-limit=0\\n" +
		"rcon.password=\\n" +
		"rcon.port=25575\\n" +
		"require-resource-pack=false\\n" +
		"resource-pack=\\n" +
		"resource-pack-prompt=\\n" +
		"resource-pack-sha1=\\n" +
		"server-ip=\\n" +
		"server-port=25565\\n" +
		"simulation-distance=10\\n" +
		"spawn-animals=true\\n" +
		"spawn-monsters=true\\n" +
		"spawn-npcs=true\\n" +
		"spawn-protection=16\\n" +
		"sync-chunk-writes=true\\n" +
		"text-filtering-config=\\n" +
		"use-native-transport=true\\n" +
		"view-distance=10\\n" +
		"white-list=false\\n"
)
