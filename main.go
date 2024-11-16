// main.go
//
// This is the entry point for the distributed cache application.
// The application uses the `discache` package to execute the main command.
//
// The `cmd.Execute()` function is called to start the application, which
// initializes and runs the distributed cache server.
//
// Usage:
//
//	To run the application, simply execute the compiled binary.
//
// Example:
//
//	$ ./discache
//
// Dependencies:
//   - github.com/dhyanio/discache/cmd: This package contains the command
//     execution logic for the distributed cache application.
package main

import (
	"github.com/dhyanio/discache/cmd"
)

func main() {
	cmd.Execute()
}
