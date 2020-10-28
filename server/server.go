package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"strings"

	shared "github.com/krithikvaidya/go_chat/shared"
)

type server struct {
	Password string

	ClientConnections map[string]net.Conn // *net.Conn?

	Address string

	connectionsMutex sync.RWMutex
}

// Initializing and return server struct
func Server(pass string, address string) *server {

	return &server{

		Password:          pass,
		ClientConnections: make(map[string]net.Conn),
		Address:           address,
	}

}

func (ser *server) listenForMessages(ctx context.Context, conn net.Conn, username string, term chan bool) {

	for {

		msg := make([]byte, 256)
		_, err := io.ReadFull(conn, msg) // read 256 bytes from client.

		if err != nil {

			if err.Error() == "EOF" {

				shared.InfoLog(fmt.Sprintf("Session terminated by client [%s], exiting...", username))
				term <- true
				return

			} else {

				shared.ErrorLog(fmt.Sprintf("Error [%s] in receiving message from [%s], continuing...", err.Error(), username))
				continue

			}

		}

		msg_str := string(msg) // Error checking?
		// shared.InfoLog(fmt.Sprintf("Got message %s", msg_str)

		f := func(c rune) bool {
			return c == '~'
		}

		msg_arr := strings.FieldsFunc(msg_str, f)

		switch msg_arr[0] {

		case "broadcast":

			msg_str = msg_arr[1]

			shared.InfoLog(fmt.Sprintf("Received broadcast message [%s] from %s", msg_str, username)) /** Messaged received will be escaped,
			  can do additional server-side validation
			*/

			to_send := fmt.Sprintf("message~%s~%s~%s~\n", username, "broadcast", msg_str)
			to_send = fmt.Sprintf("%-256v", to_send)

			// Broadcast message to all other channels
			// Obtain Read mutex

			ser.connectionsMutex.RLock()

			for uname, connxn := range ser.ClientConnections {

				if uname == username { // Don't write message back to sender
					continue
				}

				// can lead to situation where 2 separate threads write to the
				// same channel at the same time, jumbling up the messages?
				connxn.Write([]byte(to_send))

			}

			ser.connectionsMutex.RUnlock()

			shared.InfoLog(fmt.Sprintf("Successfully broadcasted message [%s] from %s", msg_str, username))

		case "pm":

			recipient := msg_arr[1]
			msg_str = msg_arr[2]

			shared.InfoLog(fmt.Sprintf("Received message [%s] from %s to send to %s", msg_str, username, recipient))

			to_send := fmt.Sprintf("message~%s~%s~%s~\n", username, "unicast", msg_str)
			to_send = fmt.Sprintf("%-256v", to_send)

			ser.connectionsMutex.RLock()

			// TODO: check if recipient exists
			connxn := ser.ClientConnections[recipient]
			connxn.Write([]byte(to_send))

			ser.connectionsMutex.RUnlock()

			shared.InfoLog(fmt.Sprintf("Successfully unicasted message [%s] from %s to %s", msg_str, username, recipient))

		case "terminate":

			shared.InfoLog(fmt.Sprintf("Recieved terminate. Shutting down connection with client %s", username))

			ser.connectionsMutex.Lock()
			defer ser.connectionsMutex.Unlock()

			delete(ser.ClientConnections, username)

			term <- true

			return

		default:

			shared.ErrorLog(fmt.Sprintf("Unexpected message type: %s", msg_arr[0]))
			// TODO: inform client
			continue

		}

	}

}

func (ser *server) handleClient(ctx context.Context, conn net.Conn) {

	// Check password. Then listen on messages.

	defer conn.Close()

	auth := make([]byte, 256)
	_, err := io.ReadFull(conn, auth) // read 256 bytes from client.

	if err != nil {
		shared.ErrorLog(fmt.Sprintf("In handleClient, %s", err.Error()))
		return
	}

	auth_str := string(auth) // error checking?

	f := func(c rune) bool {
		return c == '~'
	}

	auth_arr := strings.FieldsFunc(auth_str, f)

	if auth_arr[0] != "authenticate" {

		shared.ErrorLog(fmt.Sprintf("In handleClient, Expected \"authenticate ....\", got \"%s ....\"", auth_arr[0]))
		// TODO: gracefully cancel client
		return

	}

	if auth_arr[1] != ser.Password {

		shared.ErrorLog(fmt.Sprintf("In handleClient, Expected password to be %s, got %s", ser.Password, auth_arr[1]))
		sendStr := "terminate~Incorrect Password Received!~\n"
		sendStr = fmt.Sprintf("%-256v", sendStr)
		conn.Write([]byte(sendStr))
		// TODO: cancel client
		return

	}

	username := auth_arr[2]

	// Obtain write mutex. Current implementation supports only 1 TCP connection per client
	ser.connectionsMutex.Lock()

	_, present := ser.ClientConnections[username]

	if present {

		defer ser.connectionsMutex.Unlock()
		shared.ErrorLog(fmt.Sprintf("In handleClient, %s is already connected.", username))
		// TODO: cancel client
		return

	}

	ser.ClientConnections[username] = conn

	ser.connectionsMutex.Unlock()

	send_str := "authenticated~\n"
	send_str = fmt.Sprintf("%-256v", send_str)
	conn.Write([]byte(send_str))

	// Handling incoming messages
	term_chan := make(chan bool)

	go ser.listenForMessages(ctx, conn, username, term_chan)

	for {

		// TODO: check implicit connection termination.

		select {

		case <-ctx.Done():

			shared.InfoLog(fmt.Sprintf("Server terminated, closing connection to %s...", username))

			ser.connectionsMutex.Lock()
			defer ser.connectionsMutex.Unlock()

			delete(ser.ClientConnections, username)

			return

		// case <-time.After(2 * time.Minute): // TODO: reset at every event

		// 	shared.InfoLog(fmt.Sprintf("Closing connection to %s due to timeout", username)

		// 	ser.connectionsMutex.Lock()
		// 	defer ser.connectionsMutex.Unlock()

		// 	delete(ser.ClientConnections, username)
		// 	return

		case <-term_chan:

			ser.connectionsMutex.Lock()
			defer ser.connectionsMutex.Unlock()

			delete(ser.ClientConnections, username)

			shared.InfoLog(fmt.Sprintf("Closed connection to %s (client closed).", username))
			return

		}

	}

	// control never reaches here

}

// Server keeps persistent TCP connections with all connected clients,
// until either the server is terminated or a client is terminated/loses connection.

func (ser *server) listenForConnections(ctx context.Context, newConn chan net.Conn, listener *net.TCPListener) {

	for {

		conn, err := listener.Accept()

		if err != nil {
			shared.ErrorLog(fmt.Sprintf("%s", err.Error()))
			continue // Stop listening. TODO: investigate if this can only be caused by parent function returning
		}

		shared.InfoLog(fmt.Sprintf("Accepted an incoming connection request from [%s].", conn.RemoteAddr()))

		newConn <- conn
	}

}

// Code for getting a server running
func (ser *server) Run(ctx context.Context) {

	// Derive a new context, to pass to all goroutines created for handling client connections.
	newCtx, cancel := context.WithCancel(ctx)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ser.Address)
	shared.CheckError(err)

	// Start listening for TCP connections on the given port
	listener, err := net.ListenTCP("tcp", tcpAddr)
	shared.CheckError(err)

	newConn := make(chan net.Conn)
	go ser.listenForConnections(ctx, newConn, listener)

	// Listen and handle new connections from prospective clients...
	for {

		select {

		// case <-time.After(time.Minute):
		// timeout branch, no connection for a minute

		case <-ctx.Done(): // Context cancelled from main.go
			cancel() // Propogate cancel to all spawned goroutines
			shared.InfoLog("Server connection handler received cancel request.")
			return

		case new_conn := <-newConn:

			go ser.handleClient(newCtx, new_conn)

		}

	}

	// TODO: wait until all child goroutines exit?

}
