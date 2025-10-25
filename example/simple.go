package main

import (
	"context"
	"fmt"

	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/gogger"
)

func main() {
	log, err := gogger.NewLogger("discache.log", gogger.INFO)
	if err != nil {
	}

	SendStuff(log)
}

// SendStuff sends key-value pairs to the server
func SendStuff(log *gogger.Logger) {
	client, err := client.New("127.0.0.1:9081", client.Options{Log: log})
	if err != nil {
		log.Fatal().Msgf("Failed to create client: %v", err)
	}
	defer client.Close()

	key := []byte("foo")
	value := []byte("bar")
	ttl := 60 // TTL in seconds

	if err := client.Put(context.Background(), key, value, ttl); err != nil {
		log.Printf("Failed to put key: %v", err)
	}
	fmt.Println("Finish")
	// values, err := client.Get(context.Background(), key)
	// if err != nil {
	// 	log.Printf("Failed to get key: %v", err)
	// }
	// fmt.Printf("Value: %s", string(values))

}
