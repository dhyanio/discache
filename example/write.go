package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/gogger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./examples/local-test <host:port>")
		os.Exit(1)
	}

	endpoint := os.Args[1]

	logger, err := gogger.NewLogger("local-test.log", gogger.INFO)
	if err != nil {
		panic(err)
	}

	c, err := client.New(endpoint, client.Options{Log: logger})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx := context.Background()

	key := []byte("user:1001")
	value := []byte("Ajay")

	fmt.Println("Writing to:", endpoint)

	if err := c.Put(ctx, key, value, 0); err != nil {
		panic(err)
	}
	fmt.Println("PUT successful")

	got, err := c.Get(ctx, key)
	if err != nil {
		panic(err)
	}

	fmt.Printf("GET result: %s\n", got)

	if string(got) != string(value) {
		panic("value mismatch")
	}

	fmt.Println("TEST PASSED")
}
