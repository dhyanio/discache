package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/dhyanio/discache/cache"
)

func main() {
	opts := ServerOpts{
		ListenAddr: ":4000",
		IsLeader:   true,
	}

	go func() {
		time.Sleep(time.Second * 3)
		conn, err := net.Dial("tcp", ":4000")
		if err != nil {
			log.Fatal(err)
		}
		conn.Write([]byte("SET Foo Bar 25000000000000"))
		buf := make([]byte, 1000)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(buf[:n]))

		time.Sleep(time.Second * 2) // will not work with 5 seconds
		conn.Write([]byte("GET Foo"))

		n, err = conn.Read(buf)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(buf[:n]))
	}()

	server := NewServer(opts, cache.NewCache())
	server.Start()
}
