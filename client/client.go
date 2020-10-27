package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
)

func CheckError(err error) {

	if err != nil {
		log.Printf("<<Error>>: %s", err.Error())
		os.Exit(1)
	}

}

type client struct {
	Username       string
	ServerPassword string
	ServerHost     string
}

type message_struct struct {
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

func (cli *client) listenForServerMessages(ctx context.Context, conn net.Conn, msg_channel chan message_struct, term_chan chan bool) {

	for {

		message := make([]byte, 256)
		_, err := io.ReadFull(conn, message) // Receive message from server

		if err != nil {

			if err.Error() == "EOF" {

				log.Printf("<<Debug>>: Session terminated by server, exiting...")
				term_chan <- true
				return

			} else {

				log.Printf("<<Error>>: Error [%s] in receiving message from server, continuing...", err.Error())
				continue

			}

		}

		msg_str := string(message) // error checking?

		f := func(c rune) bool {
			return c == '~'
		}

		msg_arr := strings.FieldsFunc(msg_str, f) // supports single word messages

		msg_arr[2] = strings.Replace(msg_arr[2], "&tld;", "~", -1) // replace escaped tilde with normal tilde

		msg_channel <- message_struct{
			Sender:  msg_arr[1],
			Message: msg_arr[2],
		}

	}

}

func (cli *client) listenForClientMessages(sc bufio.Scanner, conn net.Conn) {

	for {
		if sc.Scan() { // user entered a message to send

			// TODO: check size to prevent overflow
			msg_to_send := sc.Text()
			msg_to_send = strings.Replace(msg_to_send, "~", "&tld;", -1) // replace all occurences of ~

			msg_to_send = "send~" + msg_to_send + "~\n"
			msg_to_send = fmt.Sprintf("%-256v", msg_to_send)

			bytes_sent, err := conn.Write([]byte(msg_to_send))

			if err != nil {
				log.Printf("<<Error>>: %s", err.Error())
				continue // continue the loop
			}

			log.Printf("<<Debug>>: Sent %v bytes.", bytes_sent)

			// TODO: wait for ack?
		}
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

func (cli *client) Run(ctx context.Context) {

	// Connect to the server via a raw socket

	// Check if host string is resolvable into tcp address
	serverTcpAddr, err := net.ResolveTCPAddr("tcp", cli.ServerHost) // Change to tcp4 if reqd
	CheckError(err)

	conn, err := net.DialTCP("tcp", nil, serverTcpAddr) // params are net, local address and remote address.
	CheckError(err)

	// Successfully connected to server. Now authenticate

	if cli.Username == "" {
		cli.Username = randSeq(10)
		log.Printf("\nuname is %s\n", cli.Username)
	}

	auth_str := fmt.Sprintf("authenticate~%s~%s~\n", cli.ServerPassword, cli.Username)
	auth_str = fmt.Sprintf("%-256v", auth_str)

	bytes_sent, err := conn.Write([]byte(auth_str))

	CheckError(err)
	log.Printf("<<Debug>>: Sent %v bytes.", bytes_sent)

	// TODO: set timeout

	msg := make([]byte, 256)
	_, err = io.ReadFull(conn, msg) // read 256 bytes from server

	CheckError(err)

	msg_str := string(msg) // error checking?
	msg_arr := strings.Split(msg_str, "~")

	if msg_arr[0] == "terminate" {
		log.Printf("<<Error>>: Authentication failed. %s", msg_arr[1])
		return
	}

	log.Printf("<<Debug>>: Successfully authenticated.")

	// Now let client send messages
	sc := bufio.NewScanner(os.Stdin)
	sc.Split(bufio.ScanLines)

	// Listen for messages
	msg_chan := make(chan message_struct)
	term_chan := make(chan bool)

	go cli.listenForServerMessages(ctx, conn, msg_chan, term_chan)
	go cli.listenForClientMessages(*sc, conn)

	for {

		// TODO: check for termination

		select { // select supports only channels in the cases.

		// case <-client.Context().Done():
		// DebugLogf("client send loop disconnected")

		// Don't think this works, so doing it using a separate goroutine + channel
		// case message, err := ioutil.ReadFull(conn, 256):  // Receive message from server
		case rcvd_msg := <-msg_chan:

			log.Printf("%s says: %s", rcvd_msg.Sender, rcvd_msg.Message)

		case <-term_chan:
			return

		}
	}

}
