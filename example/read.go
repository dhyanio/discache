package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/gogger"
)

func main() {
	endpoint := os.Args[1]

	logger, _ := gogger.NewLogger("reader.log", gogger.INFO)

	c, err := client.New(endpoint, client.Options{Log: logger})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	value, err := c.Get(context.Background(), []byte("user:1001"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read from %s: %s\n", endpoint, value)
}
