package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	PORT          = "6969"
	CONNECTED     = 0
	DISCONNECTED  = 1
	NEWMSG        = 2
	STRIKE_COUNT  = 3
	COMMAND       = 4
	KILL_SIGNAL   = 5
	BAN_LIMIT     = 10.0
	TOKEN_KEYWORD = "Token"
)

var (
	EOF        = errors.New("EOF")
	RATE_LIMIT = time.Duration.Seconds(4)
)

type MessageType int
type USR_ID string

type HashMap[T interface{}] struct {
	locker *sync.RWMutex
	store  map[USR_ID]T
}

func NewHashMap[T interface{}]() *HashMap[T] {
	return &HashMap[T]{
		store:  make(map[USR_ID]T),
		locker: &sync.RWMutex{},
	}
}

func (h *HashMap[T]) push(K USR_ID, V T) {
	h.locker.Lock()
	defer h.locker.Unlock()

	h.store[K] = V
}

func (h *HashMap[T]) get(K USR_ID) T {
	h.locker.RLock()
	defer h.locker.RUnlock()

	return h.store[K]
}

func (h *HashMap[T]) getAll() map[USR_ID]T {
	return h.store
}

type Client struct {
	conn        net.Conn
	lastMsg     time.Time
	strikeCount int
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

	token := genToken()
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

	clients := NewHashMap[Client]()
	banned := NewHashMap[time.Time]()
	for {
		msg := <-msgs
		UID := USR_ID(msg.Conn.RemoteAddr().String())

		switch msg.Type {
		case CONNECTED:
			bannedAt, banned := banned.store[UID]
			now := time.Now()

			if banned {
				if now.Sub(bannedAt).Seconds() >= BAN_LIMIT {
					delete(clients.store, UID)
					banned = false
				} else {
					msg.Conn.Write([]byte(fmt.Sprintf("You're banned Brah, left %v\n", BAN_LIMIT-now.Sub(bannedAt).Seconds())))
					msg.Conn.Close()
				}
			}

			if !banned {
				log.Printf("%s connected\n", UID)
				clients.push(UID, Client{conn: msg.Conn})
				msg.Conn.Write([]byte("Wellcome to server Brah!\n"))
			}

		case NEWMSG:
			client := clients.get(UID)
			client.lastMsg = time.Now()
			log.Printf("the msg sent by %s", UID)
			for id, client := range clients.store {
				if id != msg.USR_ID {
					client.conn.Write(msg.Content)
				}
			}

		case COMMAND:
			cmds := strings.Split(string(msg.Content), " ")
			cmd := cmds[0]

			if cmd == "ban" {
				ID := USR_ID(cmds[1])
				delete(clients.store, ID)
			}
		case DISCONNECTED:
			delete(clients.store, UID)
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
				log.Printf("Client disconnected\n")
				conn.Close()
				break
			}
			log.Printf("could not read the msg %s\n", err)
			conn.Close()
			break
		}

		exclamation := string(bytes[0])
		if exclamation == "!" {
			//NOTE: COMMAND SECTION;
			//checking message not empty
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

func genToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
