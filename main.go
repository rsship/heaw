package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rship/heaw/internal"
)

const (
	PORT          = "6969"
	BAN_LIMIT     = 10.0
	MESSAGE_RATE  = 1.0
	TOKEN_KEYWORD = "Token"
)

const (
	CONNECTED = iota + 1
	DISCONNECTED
	NEWMSG
	COMMAND
	KILL_SIGNAL
)

var (
	EOF = errors.New("EOF")
)

type MessageType int
type USR_ID string

type Client struct {
	Conn        net.Conn
	LastMsg     time.Time
	StrikeCount int
}

type Message struct {
	Type    MessageType
	Conn    net.Conn
	USR_ID  USR_ID
	Content []byte
}

var Dev string

func init() {
	flag.StringVar(&Dev, "HOST", ":", "DEV USAGE")
	flag.Parse()
}

func main() {
	ln, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatalf("could not start tpc connection")
		os.Exit(1)
	}

	log.Println(fmt.Sprintf("connected to TPC at %s", PORT))

	token := internal.GenToken()
	tokenchan := make(chan string)

	msgs := make(chan Message, 16)

	go server(msgs, token)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(fmt.Sprintf("could not connect to %s stream", PORT))
			msgs <- Message{
				Type: DISCONNECTED,
				Conn: conn,
			}
			conn.Close()
		}

		go client(conn, msgs, tokenchan)

		go func() {
			takenToken := <-tokenchan
			if takenToken != token {
				conn.Write([]byte(fmt.Sprintln("Wrong token Buddy nice try")))
				msgs <- Message{
					Type: DISCONNECTED,
					Conn: conn,
				}
			}
			msgs <- Message{
				Type: CONNECTED,
				Conn: conn,
			}
		}()

	}
}

func server(msgs <-chan Message, token string) {

	fmt.Printf("%s: %s\n", TOKEN_KEYWORD, token)

	clients := internal.NewInternalStore[USR_ID, Client]()
	banned := internal.NewInternalStore[USR_ID, time.Time]()

	for {
		msg := <-msgs
		UID := USR_ID(msg.Conn.RemoteAddr().String())

		switch msg.Type {
		case CONNECTED:
			bannedAt, _ := banned.Get(UID)
			now := time.Now()

			// IF ITS BANNED
			if bannedAt != nil {
				if now.Sub(*bannedAt).Seconds() >= BAN_LIMIT {
					clients.Delete(UID)
				} else {
					msg.Conn.Write([]byte(fmt.Sprintf("You're banned Brah, left %v\n", BAN_LIMIT-now.Sub(*bannedAt).Seconds())))
					msg.Conn.Close()
				}
			}
			// IF ITS NOT BANNED
			if bannedAt == nil {
				log.Printf("%s connected\n", UID)
				clients.Set(UID, Client{Conn: msg.Conn, LastMsg: now})
				msg.Conn.Write([]byte("Wellcome to server Brah!\n"))
			}

		case NEWMSG:
			client, _ := clients.Get(UID)
			if client != nil {
				now := time.Now()
				if now.Sub(client.LastMsg).Seconds() >= MESSAGE_RATE {
					if utf8.ValidString(string(msg.Content)) {
						client.LastMsg = now

						log.Printf("the msg sent by %s", UID)

						clients.ForEach(func(id USR_ID, client Client) {
							//note; do not send msg himself/herself;
							if id != msg.USR_ID {
								client.Conn.Write(msg.Content)
							}
						})

					} else {
						client.StrikeCount += 1
						if client.StrikeCount >= BAN_LIMIT {
							banned.Set(UID, now)
							msg.Conn.Write([]byte("You're banned Brah\n"))
							msg.Conn.Close()
						}
					}
				} else {
					client.StrikeCount += 1
					client.Conn.Write([]byte("Brah, Slow Down, Just Relax\n"))
					if client.StrikeCount >= BAN_LIMIT {
						banned.Set(UID, now)
						msg.Conn.Write([]byte("You're banned Brah\n"))
						msg.Conn.Close()
					}
				}
			} else {
				msg.Conn.Close()
			}

		case COMMAND:
			cmds := strings.Split(string(msg.Content), " ")
			cmd := cmds[0]

			if cmd == "ban" {
				ID := USR_ID(cmds[1])
				clients.Delete(ID)
			}
		case DISCONNECTED:
			clients.Delete(UID)
			msg.Conn.Close()
			log.Println("disconnected from server")

		}
	}
}

func client(conn net.Conn, msgs chan<- Message, token chan<- string) {
	bytes := make([]byte, 64)

	_, err := conn.Write([]byte("Paste Token Below\n"))
	if err != nil {
		fmt.Printf("couldnt write the client: %v\n", err)
		conn.Close()
	}

	n, err := conn.Read(bytes)
	if err != nil {
		fmt.Printf("couldnt read the prompt: %v\n", err)
		conn.Close()
	}

	tempToken := strings.TrimSpace(string(bytes[:n]))
	if strings.Contains(tempToken, TOKEN_KEYWORD) {
		tempToken = strings.TrimSpace(strings.Split(tempToken, fmt.Sprintf("%s: ", TOKEN_KEYWORD))[1])
	}

	//NOTE: wait for token verification
	token <- tempToken

	for {
		n, err := conn.Read(bytes)
		if err != nil {
			//NOTE: checking EOF exception;
			if err.Error() == EOF.Error() {
				log.Printf("could not read the msg %s\n", err)
				conn.Close()
				break
			}

			log.Printf("Client disconnected\n")
			conn.Close()
			break
		}

		exclamation := string(bytes[0])
		if exclamation == "!" {
			//NOTE: COMMAND SECTION;
			if n > 0 {
				msgs <- Message{
					Type:    COMMAND,
					Conn:    conn,
					Content: bytes[1:n],
					USR_ID:  USR_ID(conn.RemoteAddr().String()),
				}

			}
			continue
		}

		msgs <- Message{
			Type:    NEWMSG,
			Conn:    conn,
			Content: bytes[:n],
			USR_ID:  USR_ID(conn.RemoteAddr().String()),
		}
	}
}
