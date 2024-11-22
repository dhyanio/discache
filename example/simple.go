package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/gogger"
)

func main() {
	// Initialize Logger
	logFile, err := os.Create("discache.log")
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	log, err := gogger.NewLogger("discache.log", gogger.INFO)
	if err != nil {
	}

	SendStuff(log)
}

// SendStuff sends key-value pairs to the server
func SendStuff(log *gogger.Logger) {
	client, err := client.New(":3000", client.Options{Log: log})
	if err != nil {
		log.Fatal().Msgf("Failed to create client: %v", err)
	}
	defer client.Close()

	for i := 0; i < 70; i++ {
		key := []byte(fmt.Sprintf("test_key_%d", i))
		value := []byte(fmt.Sprintf("test_value_%d", i))

		err = client.Put(context.Background(), key, value, 0)
		if err != nil {
			log.Fatal().Msgf("Failed to put key-value pair: %v", err)
		}

		fmt.Println("GET", string(key))
		resp, err := client.Get(context.Background(), key)
		if err != nil {
			log.Fatal().Msgf("Failed to get value: %v", err)
		}
		fmt.Println(string(resp))
	}
}
