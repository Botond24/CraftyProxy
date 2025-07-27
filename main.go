package main

import (
	"crypto/tls"
	"github.com/Botond24/CraftyProxy/crafty"
	"github.com/Botond24/CraftyProxy/proxy"
	"net/http"
	"sync"
)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conf := getConfig()
	c := crafty.New(conf.CraftyAddr, conf.Port, conf.Key, conf.Timeout)
	c.GetServers()
	var wg sync.WaitGroup
	for _, server := range c.Servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy.Handle(&server, conf.Addr)
		}()
	}
	wg.Wait()
}
