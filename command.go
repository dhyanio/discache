package main

import "time"


type Command string

cosnt (
	CMDSet command = "SET"
	CMDGet command = "GET"
)

type Message struct {
	Cmd Command
	Key []byte
	Value []byte
	TTL time.Duration
}