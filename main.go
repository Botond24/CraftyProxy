package main

import (
	"CraftyProxy/crafty"
	"CraftyProxy/proxy"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"sync"
)

func main() {
	log.New(os.Stdout, "", 0)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	c := crafty.New("10.0.0.12", 8443, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpYXQiOjE3NTM0NTk1NDIsInRva2VuX2lkIjoyfQ.VCcf1TtuYCA7WrVL6RxGUqEvMkHI1_HT-BSR2GDomWA", 5)
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
