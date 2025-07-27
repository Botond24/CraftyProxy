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
	c := crafty.New(conf.CraftyAddr, conf.Port, conf.Key, conf.Timeout)
	c.GetServers()
	var wg sync.WaitGroup
	for _, server := range c.Servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy.Handle(&server,conf.Addr)
		}()
	}
	wg.Wait()
}
