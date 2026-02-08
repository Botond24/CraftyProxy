package main

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/Botond24/CraftyProxy/crafty"
	"github.com/Botond24/CraftyProxy/proxy"
)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conf := getConfig()
	c := crafty.New(conf.CraftyAddr, conf.Port, conf.Key, conf.Timeout)
	c.GetServers()
	var wg sync.WaitGroup
	go c.ListenWs(&wg, proxy.Handle)
	for _, server := range c.Servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy.Handle(&server, conf.Addr)
		}()
	}
	wg.Wait()
}
