package main

/**

Protocol for sending/receiving messages:
- Each message will be of length 256 bytes.
- There will be 4 types of messages:

1. authenticate <server_password> <client_username>  --> sent from client to server
2. authenticated  --> sent from server to client
3. send <message>  --> from client to server
4. message <username> <message>  --> broadcasted from server to all other clients.
5. terminate <reason>  --> from server to client, for graceful shutdown

*/

/**
TODO:
use panic for errors
add support for utf-8 https://medium.com/rungo/string-data-type-in-go-8af2b639478
add prompt for entering a message
stuff labelled as todo
check if contexts are propogated properly
dockerize it  -- DONE

Possible extensions:
use gorilla websockets instead of raw sockets, get a web-based frontend
use RPCs for communication instead of raw sockets
persist messages
allow for 1:1 communication (should be easy)
write tests
add security
allow for variable size messages
deploy to heroku

*/

import (
	"flag"
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"log"
	"os"
	"os/signal"
	"syscall"

	client "github.com/krithikvaidya/go_chat/client"
	server "github.com/krithikvaidya/go_chat/server"
)

var (
	serverMode bool
	host       string
	password   string
	username   string
)

func init() {

	// Implement command-line flag parsing
	flag.BoolVar(&serverMode, "s", false, "to run as the server")
	flag.StringVar(&host, "h", "0.0.0.0:4545", "the chat server's host:port")
	flag.StringVar(&password, "p", "", "the chat server's password")
	flag.StringVar(&username, "n", "", "client username")
	flag.Parse()

	// log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano()) // For generating random username

}

func CheckError(err error) {

	if err != nil {
		log.Printf("<<Error>>: %s", err.Error())
		os.Exit(1)
	}

}

func main() {

	base_ctx := context.Background()            // Creating an empty context
	ctx, cancel := context.WithCancel(base_ctx) // Creating a cancellable context

	os_sigs := make(chan os.Signal, 1)                      // Listen for OS signals, with buffer size 1
	signal.Notify(os_sigs, syscall.SIGTERM, syscall.SIGINT) // SIGKILL, SIGQUIT?

	// Listen for shutdown signals on a separate thread
	go func() {

		format := "Listening for shutdown signal..."
		log.Printf("<<Debug>>: " + format)

		rcvd_sig := <-os_sigs // Wait till a SIGINT or a SIGQUIT is received

		format = "Signal received"
		log.Printf("<<Debug>>: %v: %v", format, rcvd_sig)

		signal.Stop(os_sigs) // Stop listening for signals
		close(os_sigs)

		cancel()

		os.Exit(0) // Exit entire program from a goroutine

	}()

	var err error

	if serverMode {

		log.Printf("<<Debug>>: Running in server mode...")
		server.Server(password, host).Run(ctx) // Passing ctx should trigger a Done signal in Run() when a
		// shutdown signal is encountered above.

	} else {

		log.Printf("<<Debug>>: Running in client mode...")
		client.Client(password, host, username).Run(ctx)

	}

	CheckError(err)

}
