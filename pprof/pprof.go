package main

import (
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go http.ListenAndServe("localhost:6060", nil)

	select {}
}
