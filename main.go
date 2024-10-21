package main

import (
	"log"
	"net"
	"time"

	"github.com/dhyanio/discache/cache"
)

func main() {
	opts := ServerOpts{
		ListenAddr: ":3000",
		IsLeader:   true,
	}

	go func() {
		time.Sleep(time.Second * 2)
		conn, err := net.Dial("tcp", ":3000")
		if err != nil {
			log.Fatal(err)
		}
		conn.Write([]byte("SET Foo Bar 2500"))
	}()

	server := NewServer(opts, cache.NewCache())
	server.Start()
}
