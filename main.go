package main

import (
	"context"
	"flag"
	"log"
	"net"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/client"
)

// func third() {
// 	go func() {
// 		time.Sleep(time.Second * 3)

// 		buf := make([]byte, 1000)
// 		n, err := conn.Read(buf)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		fmt.Println(string(buf[:n]))

// 		time.Sleep(time.Second * 2) // will not work with 5 seconds
// 		conn.Write([]byte("GET Foo"))

// 		n, err = conn.Read(buf)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		fmt.Println(string(buf[:n]))
// 	}()

// }

func m() {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Write([]byte("SET Foo Bar 4000000"))
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

func main() {
	// m()
	var (
		listenAddr = flag.String("listenaddr", ":3000", "listen address fo the server")
		leaderAddr = flag.String("leaderaddr", "", "listen address of the leader")
	)

	flag.Parse()
	opts := ServerOpts{
		ListenAddr: *listenAddr,
		IsLeader:   len(*leaderAddr) == 0,
		LeaderAddr: *leaderAddr,
	}

	go func() {
		time.Sleep(time.Second * 2)
		client, err := client.New(":3000", client.Options{})
		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < 10; i++ {
			SendCommand(client)
		}
		client.Close()
		time.Sleep(time.Second * 1)
	}()

	server := NewServer(opts, cache.NewCache())
	server.Start()
}

func SendCommand(c *client.Client) {
	_, err := c.Set(context.Background(), []byte("Foo"), []byte("Bar"), 0)
	if err != nil {
		log.Fatal(err)
	}

}
