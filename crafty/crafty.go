package crafty

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Crafty struct {
	ip          string
	url         string
	Key         string
	Servers     []Server
	logger      *log.Logger
	StopTimeout time.Duration
}

type serversResponse struct {
	Data []jsonServer `json:"data"`
}

type jsonServer struct {
	Name string `json:"server_name"`
	Id   string `json:"server_id"`
	Ip   string `json:"server_ip"`
	Port int    `json:"server_port"`
}

func New(address string, port int, key string, timeout int) *Crafty {
	c := new(Crafty)
	c.url = "https://" + address + ":" + strconv.Itoa(port)
	c.Key = key
	c.Servers = []Server{}
	c.logger = log.New(os.Stdout, "crafty("+address+"): ", log.Ldate|log.Ltime)
	c.StopTimeout = time.Duration(timeout)
	c.ip = address
	return c
}

func (c *Crafty) Get(path string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.url+path, nil)
	if err != nil {
		c.logger.Println("Can't create request: " + err.Error() + "\n")
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	return client.Do(req)
}

func (c *Crafty) Post(path string, data []byte) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", c.url+path, strings.NewReader(string(data)))
	if err != nil {
		c.logger.Println("Can't create request: " + err.Error() + "\n")
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	return client.Do(req)
}

func (c *Crafty) Patch(path string, data []byte) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PATCH", c.url+path, strings.NewReader(string(data)))
	if err != nil {
		c.logger.Println("Can't create request: " + err.Error() + "\n")
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	return client.Do(req)
}

func (c *Crafty) Put(path string, data []byte) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", c.url+path, strings.NewReader(string(data)))
	if err != nil {
		c.logger.Println("Can't create request: " + err.Error() + "\n")
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	return client.Do(req)
}

func (c *Crafty) GetServers() {
	path := "/api/v2/servers"
	resp, err := c.Get(path)
	if err != nil {
		panic("Request failed: " + err.Error() + "\n")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Can't read response body: " + err.Error() + "\n")
	}

	var servers serversResponse
	err = json.Unmarshal(body, &servers)
	if err != nil {
		panic("Can't extract JSON to object: " + err.Error() + "\n")
	}
	for _, server := range servers.Data {
		s := NewServer(c, server)
		c.Servers = append(c.Servers, *s)
	}
	c.Servers = filter(c.Servers, func(server Server) bool {
		return server.AutoOn || server.autoOff
	})
	c.logger.Println("Found " + strconv.Itoa(len(c.Servers)) + " servers")
}

func filter[T any](s []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(s)) // Pre-allocate for efficiency
	for _, v := range s {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}
