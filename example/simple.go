package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/discache/logger"
)

func main() {
	// Initialize Logger
	logFile, _ := os.Create("discache.log")
	defer logFile.Close()

	log := logger.NewLogger(logger.INFO, logFile)

	SendStuff(log)
}

func SendStuff(log *logger.Logger) {
	client, err := client.New(":3000", client.Options{Log: log})
	if err != nil {
		log.Fatal(err.Error())
	}

	for i := 0; i < 70; i++ {
		var (
			key   = []byte(fmt.Sprintf("test_key_%d", i))
			value = []byte(fmt.Sprintf("test_value_%d", i))
		)
		err = client.Put(context.Background(), key, value, 0)
		if err != nil {
			log.Fatal(err.Error())
		}

		fmt.Println("GET", string(key))
		resp, err := client.Get(context.Background(), key)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Info(string(resp))
	}

	client.Close()
}
