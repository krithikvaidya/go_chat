package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	shared "github.com/krithikvaidya/go_chat/shared"
)

type client struct {
	Username       string
	ServerPassword string
	ServerHost     string
}

type message_struct struct {
	Type    int
	Message string
	Sender  string
}

func Client(password string, host string, username string) *client {

	return &client{

		Username:       username,
		ServerPassword: password,
		ServerHost:     host,
	}

}

func getServerMessage(conn net.Conn, rcvd_msg chan string, exit_chan chan bool) {

	// defer func() {
	// 	shared.InfoLog("\n\n0 --- DONE\n\n")
	// }()

	for {
		message := make([]byte, 256)
		_, err := io.ReadFull(conn, message) // Receive message from server

		if err != nil {

			if err.Error() == "EOF" {

				shared.ErrorLog("Session terminated by server, exiting...")
				defer func(exit_chan chan bool) {
					exit_chan <- true // will not block
				}(exit_chan)
				return

			} else {

				shared.ErrorLog(fmt.Sprintf("Error [%s] in receiving message from server, continuing...", err.Error()))
				continue

			}
		}

		rcvd_msg <- string(message)

	}

}

func (cli *client) listenForServerMessages(ctx context.Context, conn net.Conn, msg_channel chan message_struct, term_chan chan bool, final_term_chan chan bool) {

	// defer func() {
	// 	shared.InfoLog("\n\n1 --- DONE\n\n")
	// }()

	rcvd_msg := make(chan string)
	exit_chan := make(chan bool)

	go getServerMessage(conn, rcvd_msg, exit_chan)
	msg_str := ""

	for {

		select {

		case <-time.After(time.Second):
			continue

		case msg_str = <-rcvd_msg:
			// Do nothing

		case <-exit_chan:
			defer func(final_term_chan, term_chan chan bool) {

				term_chan <- true
				final_term_chan <- true

			}(final_term_chan, term_chan)
			return

		case <-ctx.Done():
			return

		}

		f := func(c rune) bool {
			return c == '~'
		}

		msg_arr := strings.FieldsFunc(msg_str, f)
		msg_type := 0 // 0 indicates broadcast

		if msg_arr[2] == "unicast" {
			msg_type = 1
		}

		msg_arr[3] = strings.Replace(msg_arr[3], "&tld;", "~", -1) // replace escaped tilde with normal tilde

		msg_channel <- message_struct{
			Type:    msg_type,
			Sender:  msg_arr[1],
			Message: msg_arr[3],
		}

	}

}

func getClientMessage(sc bufio.Scanner, conn net.Conn, rcvd_msg chan string) {

	defer func() {
		shared.InfoLog("\n\n2 --- DONE\n\n")
	}()

	for {
		msg_rcvd := ""

		if sc.Scan() { // user entered a message to send

			// TODO: check size to prevent overflow
			msg_rcvd = sc.Text()

		} else {
			break
		}

		rcvd_msg <- msg_rcvd
	}

}

func (cli *client) listenForClientMessages(ctx context.Context, sc bufio.Scanner, conn net.Conn, final_term_chan chan bool) {

	// defer func() {
	// 	shared.InfoLog("\n\n3 --- DONE\n\n")
	// }()

	message := ""
	rcvd_msg := make(chan string)

	go getClientMessage(sc, conn, rcvd_msg)

	for {

		select {

		case <-time.After(time.Second):
			continue

		case message = <-rcvd_msg:
			// Do nothing

		case <-ctx.Done():

			defer func(final_term_chan chan bool) {
				final_term_chan <- true // will be received immediately
			}(final_term_chan)
			return

		}

		msg_rcvd := strings.Replace(message, "~", "&tld;", -1) // replace all occurences of ~

		// TODO: assume bad formatting of message is possible
		first_space := strings.IndexByte(msg_rcvd, ' ')
		second_space := first_space + 1 + strings.IndexByte(msg_rcvd[first_space+1:], ' ')

		command := msg_rcvd[:first_space]

		msg_to_send := ""

		if command == "/broadcast" {

			msg_to_send = "broadcast~" + msg_rcvd[first_space+1:] + "~\n"

		} else { // /pm

			recipient := msg_rcvd[first_space+1 : second_space]
			msg := msg_rcvd[second_space+1:]
			msg_to_send = "pm~" + recipient + "~" + msg + "~\n"

		}

		msg_to_send = fmt.Sprintf("%-256v", msg_to_send)

		bytes_sent, err := conn.Write([]byte(msg_to_send))

		if err != nil {
			shared.InfoLog(fmt.Sprintf("%s", err.Error()))
			continue // continue the loop
		}

		shared.InfoLog(fmt.Sprintf("Sent %v bytes.", bytes_sent))

	}

}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (cli *client) Run(ctx context.Context, main_term_chan chan bool) {

	// defer func() {
	// 	shared.InfoLog("\n\n4 --- DONE\n\n")
	// }()

	// Connect to the server via a raw socket

	// Check if host string is resolvable into tcp address
	serverTcpAddr, err := net.ResolveTCPAddr("tcp", cli.ServerHost) // Change to tcp4 if reqd
	shared.CheckError(err)

	conn, err := net.DialTCP("tcp", nil, serverTcpAddr) // params are net, local address and remote address.
	shared.CheckError(err)

	// Successfully connected to server. Now authenticate

	if cli.Username == "" {
		cli.Username = randSeq(10)
		shared.InfoLog(fmt.Sprintf("\nYour randomly generated username is: %s\n", cli.Username))
	}

	auth_str := fmt.Sprintf("authenticate~%s~%s~\n", cli.ServerPassword, cli.Username)
	auth_str = fmt.Sprintf("%-256v", auth_str)

	bytes_sent, err := conn.Write([]byte(auth_str))

	shared.CheckError(err)
	shared.InfoLog(fmt.Sprintf("Sent %v bytes.", bytes_sent))

	// TODO: set timeout

	msg := make([]byte, 256)
	_, err = io.ReadFull(conn, msg) // read 256 bytes from server

	shared.CheckError(err)

	msg_str := string(msg) // error checking?
	msg_arr := strings.Split(msg_str, "~")

	if msg_arr[0] == "terminate" {
		shared.ErrorLog(fmt.Sprintf("Authentication failed. %s", msg_arr[1]))
		return
	}

	shared.InfoLog("Successfully authenticated.")
	shared.InfoLog("Message format: /pm <username> <message> or /broadcast <message>")

	// Now let client send messages
	sc := bufio.NewScanner(os.Stdin)
	sc.Split(bufio.ScanLines)

	// Listen for messages
	msg_chan := make(chan message_struct)
	term_chan := make(chan bool)
	final_term_chan := make(chan bool)

	// Derive new context
	newCtx, cancel := context.WithCancel(ctx)

	go cli.listenForServerMessages(newCtx, conn, msg_chan, term_chan, final_term_chan)
	go cli.listenForClientMessages(newCtx, *sc, conn, final_term_chan)

	for {

		// TODO: check for termination

		select { // select supports only channels in the cases.

		case rcvd_msg := <-msg_chan:

			msg_type := "broadcast"

			if rcvd_msg.Type == 1 {
				msg_type = "pm"
			}

			shared.MessageLog(rcvd_msg.Sender, msg_type, rcvd_msg.Message)

		case <-term_chan:

			cancel()

			<-final_term_chan
			<-final_term_chan

			// defer func(main_term_chan chan bool) {

			// }(main_term_chan)

			return

		case <-ctx.Done():
			shared.InfoLog("Received termination signal, exiting...")
			cancel()
			<-final_term_chan
			<-final_term_chan
			shared.InfoLog("Exited.")
			main_term_chan <- true
			return

		}
	}

}
