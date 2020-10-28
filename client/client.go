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

func (cli *client) listenForServerMessages(ctx context.Context, conn net.Conn, msg_channel chan message_struct, term_chan chan bool) {

	for {

		message := make([]byte, 256)
		_, err := io.ReadFull(conn, message) // Receive message from server

		if err != nil {

			if err.Error() == "EOF" {

				shared.ErrorInfo("Session terminated by server, exiting...")
				term_chan <- true
				return

			} else {

				shared.ErrorInfo(fmt.Sprintf("Error [%s] in receiving message from server, continuing...", err.Error()))
				continue

			}

		}

		msg_str := string(message) // error checking?

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

func (cli *client) listenForClientMessages(sc bufio.Scanner, conn net.Conn) {

	for {
		if sc.Scan() { // user entered a message to send

			// TODO: check size to prevent overflow
			msg_rcvd := sc.Text()
			msg_rcvd = strings.Replace(msg_rcvd, "~", "&tld;", -1) // replace all occurences of ~

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
				shared.LogInfo(fmt.Sprintf("%s", err.Error()))
				continue // continue the loop
			}

			shared.LogInfo(fmt.Sprintf("Sent %v bytes.", bytes_sent))
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
		shared.LogInfo(fmt.Sprintf("\nuname is %s\n", cli.Username))
	}

	auth_str := fmt.Sprintf("authenticate~%s~%s~\n", cli.ServerPassword, cli.Username)
	auth_str = fmt.Sprintf("%-256v", auth_str)

	bytes_sent, err := conn.Write([]byte(auth_str))

	shared.CheckError(err)
	shared.LogInfo(fmt.Sprintf("Sent %v bytes.", bytes_sent))

	// TODO: set timeout

	msg := make([]byte, 256)
	_, err = io.ReadFull(conn, msg) // read 256 bytes from server

	CheckError(err)

	msg_str := string(msg) // error checking?
	msg_arr := strings.Split(msg_str, "~")

	if msg_arr[0] == "terminate" {
		shared.ErrorInfo(fmt.Sprintf("Authentication failed. %s", msg_arr[1]))
		return
	}

	shared.LogInfo("Successfully authenticated.")
	shared.LogInfo("Send a message in the following format: /pm <username> <message> or /broadcast <message>")

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

			msg_type := "broadcast"

			if rcvd_msg.Type == 1 {
				msg_type = "pm"
			}

			log.Printf("%s says (via %s): %s", rcvd_msg.Sender, msg_type, rcvd_msg.Message)

		case <-term_chan:
			return

		}
	}

}
