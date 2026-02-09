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

	"github.com/Tnze/go-mc/bot"
)

type fileResponse struct {
	Data string `json:"data"`
}

type Server struct {
	Name         string
	parent       *Crafty
	InPort       uint16
	OutPort      uint16
	AutoOn       bool
	AutoOff      bool
	ChangePort   bool
	VoiceInPort  int
	VoiceOutPort int
	id           string
	Logger       *log.Logger
	Address      string
	players      int
	stopTimer    *time.Timer
	State        string
	Handled      bool
}

func NewServer(parent *Crafty, srv jsonServer) *Server {
	s := new(Server)
	s.parent = parent

	s.AutoOn = false
	s.AutoOff = false
	s.ChangePort = false
	s.VoiceInPort = 0
	s.VoiceOutPort = 0

	s.InPort = srv.Port - 2000
	s.OutPort = srv.Port

	s.id = srv.Id
	s.players = 0
	s.Address = srv.Ip
	if srv.Ip == "127.0.0.1" {
		s.Address = parent.ip
	}
	var options []string
	s.Name, options = s.FixName(srv.Name)
	s.Logger = log.New(os.Stdout, "crafty("+parent.ip+") | "+s.Name+": ", log.Ldate|log.Ltime)

	if slices.Contains(options, "player-start") {
		s.AutoOn = true
	}
	if slices.Contains(options, "player-stop") {
		s.AutoOff = true
	}
	if slices.ContainsFunc(options, containsVoice) {
		idx := slices.IndexFunc(options, containsVoice)
		if idx == -1 {
			s.Logger.Fatalf("Invalid options: %v", options)
		}
		l := strings.Split(options[idx], "=")
		p, err := strconv.Atoi(l[1])
		if err != nil {
			s.Logger.Fatalf("Invalid options: %v", options)
		}
		if p == -1 {
			p = int(s.OutPort)
		}
		s.VoiceOutPort = p
		s.VoiceInPort = p - 2000
	}
	if slices.Contains(options, "update-port") {
		s.ChangePort = true
	}
	if s.ChangePort {
		s.updatePort()
	} else {
		s.InPort = s.OutPort
		s.VoiceInPort = s.VoiceOutPort
	}
	s.stopTimer = time.AfterFunc(parent.StopTimeout*time.Minute, func() {
		s.Stop()
	})
	s.stopTimer.Stop()
	s.IsRunning()
	return s
}
func containsVoice(str string) bool {
	return strings.Contains(str, "voice-port")
}

func (s *Server) String() string {
	return s.Name + " (" + strconv.Itoa(int(s.OutPort)) + "->" + strconv.Itoa(int(s.InPort)) + ")" + "\n" +
		"\tAuto on: " + strconv.FormatBool(s.AutoOn) + "\n" +
		"\tAuto off: " + strconv.FormatBool(s.AutoOff) + "\n" +
		"\tPorts: " + strconv.Itoa(int(s.InPort)) + " -> " + strconv.Itoa(int(s.OutPort)) + "\n" +
		"\tID: " + s.id
}

func (s *Server) Start(name string) {
	if s.State != "stopped" {
		return
	}
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/action/start_server", []byte{})
	if err != nil {
		s.Logger.Println("Can't start server: " + err.Error())
		return
	}
	if post.StatusCode != 200 {
		s.Logger.Println("Can't start server: " + post.Status)
		return
	}
	s.Logger.Println("Server started by " + name)
	s.State = "starting"
}

func (s *Server) Stop() {
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/action/stop_server", []byte{})
	if err != nil {
		s.Logger.Println("Can't stop server: " + err.Error())
		return
	}
	if post.StatusCode != 200 {
		s.Logger.Println("Can't stop server: " + post.Status)
		return
	}
	s.Logger.Println("Stopped server")
	s.State = "stopped"
}

func (s *Server) updatePort() {
	if s.AutoOn || s.AutoOff {
		s.updateServerProperties()
	}
	if s.VoiceOutPort != 0 {
		s.updateVoiceProperties()
	}
}

func (s *Server) createProperties() {
	body := "{\"parent\":\"servers/" + s.id + "\",\"name\": \"server.property\",\"directory\": false}"
	put, err := s.parent.Put("/api/v2/servers/"+s.id+"/files/servers/"+s.id+"/server.properties", []byte(body))
	if err != nil {
		s.Logger.Println("Can't create server.properties: " + err.Error() + "\n")
		return
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
		if s.checkPing() {
			s.State = "running"
			return true
		}
		return false
	}
	s.State = "stopped"
	return false
}

func (s *Server) IncrementPlayers() {
	s.players++
	s.Logger.Println("Players: " + strconv.Itoa(s.players))
	if s.AutoOff {
		s.stopTimer.Stop()
	}
}

func (s *Server) DecrementPlayers() {
	s.players--
	s.Logger.Println("Players: " + strconv.Itoa(s.players))
	if s.players <= 0 {
		s.players = 0
		if s.AutoOff {
			s.Logger.Println("Stopping server in " + strconv.Itoa(int(s.parent.StopTimeout)) + " minutes")
			s.stopTimer.Reset(s.parent.StopTimeout * time.Minute)
		}

	}
}

func (s *Server) Remove() {
	s.parent.Servers = slices.DeleteFunc(s.parent.Servers, func(server Server) bool {
		return server.id == s.id
	})
}

func (s *Server) FixName(inname string) (name string, options []string) {
	options = strings.Split(inname, "$")
	if len(options) > 1 {
		name = strings.TrimSpace(options[0])
		options = strings.Split(options[1], "&")
	} else {
		name = inname
		options = []string{}
	}
	return
}

func (s *Server) checkPing() bool {
	_, _, err := bot.PingAndList(s.Address + ":" + strconv.Itoa(int(s.InPort)))
	if err != nil {
		return false
	}
	return true
}

func (s *Server) updateServerProperties() {
	body := s.getFileDefault("server.properties", defaultServerProperties)

	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.Contains(line, "server-port=") {
			lines[i] = "server-port=" + strconv.Itoa(int(s.InPort))
		}
		if strings.Contains(line, "query.port=") {
			lines[i] = "query.port=" + strconv.Itoa(int(s.InPort))
		}
	}
	body = strings.Join(lines, "\n")
	s.updateFile("server.properties", body)
}

func (s *Server) updateVoiceProperties() {
	body := s.getFileDefault("config/voicechat/voicechat-server.properties", "port=")
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.Contains(line, "port=") {
			lines[i] = "port=" + strconv.Itoa(s.VoiceInPort)
		}
	}
	body = strings.Join(lines, "\n")
	s.updateFile("config/voicechat/voicechat-server.properties", body)
}

func (s *Server) getFileDefault(filename string, def string) string {
	path := "servers/" + s.id + "/" + filename
	props := "{\"filename\":\"" + path + "\"}"
	post, err := s.parent.Post("/api/v2/servers/"+s.id+"/files", []byte(props))
	if err != nil {
		s.Logger.Println("Can't update port: " + err.Error())
		return ""
	}
	body := def
	if post.StatusCode != 200 {
		s.Logger.Println(filename + " not found, creating...")
		s.createProperties()
	} else {
		defer post.Body.Close()
		postBody, err := io.ReadAll(post.Body)
		if err != nil {
			s.Logger.Println("Can't read response body: " + err.Error())
			return ""
		}
		var file fileResponse
		json.Unmarshal(postBody, &file)
		body = file.Data
	}
	return body
}

func (s *Server) updateFile(filename string, body string) {
	path := "servers/" + s.id + "/" + filename
	body = "{\"path\":\"" + path + "\",\"contents\":\"" + strings.ReplaceAll(
		strings.ReplaceAll(body, "\n", "\\n"), "\\:", "\\\\:") + "\"}"
	patch, err := s.parent.Patch("/api/v2/servers/"+s.id+"/files", []byte(body))
	if err != nil {
		s.Logger.Println("Can't update " + filename + ": " + err.Error())
		return
	}
	if patch.StatusCode != 200 {
		s.Logger.Println("Can't update \"+filename+\"s: " + patch.Status)
		defer patch.Body.Close()
		patchBody, err := io.ReadAll(patch.Body)
		if err != nil {
			s.Logger.Println("Can't read response body: " + err.Error())
		}
		s.Logger.Println(string(patchBody))
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
