package main

import (
	"CraftyProxy/crafty"
	"CraftyProxy/proxy"
	"crypto/tls"
	"net/http"
	"sync"
)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conf := getConfig()
	c := crafty.New(conf.Addr, conf.Port, conf.Key, conf.Timeout)
	c.GetServers()
	var wg sync.WaitGroup
	for _, server := range c.Servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy.Handle(&server)
		}()
	}
	wg.Wait()
}
