package main

/**

Protocol for sending/receiving messages:
- Each message will be of length 256 bytes.
- There will be 6 types of messages:

1. authenticate~<server_password>~<client_username>~\n  --> sent from client to server
2. authenticated~\n  --> sent from server to client
3. pm~<username>~<message>~\n  --> from client to server, unicast message
4. broadcast~<message>~\n  --> from client to server, broadcast message
5. message~<username>~<type>~<message>~\n  --> from server to client(s). Type indicates unicast/broadcast.
6. terminate~<reason>~\n  --> from server to client, for graceful shutdown

*/

/**
TODO:
add support for utf-8 https://medium.com/rungo/string-data-type-in-go-8af2b639478
add prompt for entering a message
stuff labelled as todo
check if contexts are propogated properly
dockerize it  -- DONE
allow for 1:1 communication (should be easy) -- done

Possible extensions:
use gorilla websockets instead of raw sockets, get a web-based frontend
use RPCs for communication instead of raw sockets
persist messages
write tests
add security
allow for variable size messages
deploy to heroku

*/

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/net/context"

	"log"
	"os"
	"os/signal"
	"syscall"

	client "github.com/krithikvaidya/go_chat/client"
	server "github.com/krithikvaidya/go_chat/server"
	shared "github.com/krithikvaidya/go_chat/shared"
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

	log.SetFlags(0)                  // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano()) // For generating random username

}

func main() {

	base_ctx := context.Background()            // Creating an empty context
	ctx, cancel := context.WithCancel(base_ctx) // Creating a cancellable context

	os_sigs := make(chan os.Signal, 1)                      // Listen for OS signals, with buffer size 1
	signal.Notify(os_sigs, syscall.SIGTERM, syscall.SIGINT) // SIGKILL and SIGSTOP cannot be caught by a program

	main_term_chan := make(chan bool)

	// Listen for shutdown signals on a separate thread
	go func(term_chan chan bool) {

		format := "Listening for shutdown signal..."
		shared.InfoLog(format)

		rcvd_sig := <-os_sigs // Wait till a SIGINT or a SIGQUIT is received

		format = "Signal received"
		shared.InfoLog(fmt.Sprintf("%v: %v", format, rcvd_sig))

		signal.Stop(os_sigs) // Stop listening for signals
		close(os_sigs)

		cancel()

		for {
			select {
			case <-time.After(10 * time.Second):
				shared.InfoLog("Timeout expired, force shutdown invoked.")
				break
			case <-main_term_chan:
				shared.InfoLog("Shutdown complete successfully.")
				break
			}
		}

		os.Exit(0) // Exit entire program from a goroutine

	}(main_term_chan)

	if serverMode {

		shared.InfoLog("Running in server mode...")
		server.Server(password, host).Run(ctx) // Passing ctx should trigger a Done signal in Run() when a
		// shutdown signal is encountered above.

	} else {

		shared.InfoLog("Running in client mode...")
		client.Client(password, host, username).Run(ctx, main_term_chan)
		os.Exit(0)
		shared.InfoLog("last")
	}

}
