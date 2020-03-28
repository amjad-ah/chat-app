package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/amjad-ah/chat-app/chat"
	"google.golang.org/grpc"
)

type Connection struct {
	conn chat.Chat_ChatServer
	send chan *chat.ChatMessage
	quit chan struct{}
}

type ChatServer struct {
	broadcast  chan *chat.ChatMessage
	quit       chan struct{}
	connection []*Connection
	connLock   sync.Mutex
}

func newConnection(conn chat.Chat_ChatServer) *Connection {
	c := &Connection{
		conn: conn,
		send: make(chan *chat.ChatMessage),
		quit: make(chan struct{}),
	}

	go c.start()

	return c
}

func (c Connection) Close() error {
	close(c.quit)
	close(c.send)

	return nil
}

func (c Connection) Send(msg *chat.ChatMessage) {
	defer func() {
		recover()
	}()

	c.send <- msg
}

func (c Connection) start() {
	running := true

	for running {
		select {
		case msg := <-c.send:
			c.conn.Send(msg)
		case <-c.quit:
			running = false
		}
	}
}

func (c Connection) GetMessages(broadcast chan<- *chat.ChatMessage) error {
	for {
		msg, err := c.conn.Recv()
		if err == io.EOF {
			c.Close()
			return nil
		} else if err != nil {
			c.Close()
			return err
		}

		go func(msg *chat.ChatMessage) {
			select {
			case broadcast <- msg:
			case <-c.quit:
			}
		}(msg)
	}
}

func NewChatServer() *ChatServer {
	srv := &ChatServer{
		broadcast: make(chan *chat.ChatMessage),
		quit:      make(chan struct{}),
	}

	go srv.start()
	return srv
}

func (c *ChatServer) start() {
	running := true

	for running {
		select {
		case msg := <-c.broadcast:
			c.connLock.Lock()
			for _, v := range c.connection {
				go v.Send(msg)
			}
			c.connLock.Unlock()
		case <-c.quit:
			running = false
		}
	}
}

func (c *ChatServer) Chat(stream chat.Chat_ChatServer) error {
	conn := newConnection(stream)

	c.connLock.Lock()
	c.connection = append(c.connection, conn)
	c.connLock.Unlock()

	err := conn.GetMessages(c.broadcast)

	c.connLock.Lock()
	for i, v := range c.connection {
		if v == conn {
			c.connection = append(c.connection[:i], c.connection[i+1:]...)
		}
	}
	c.connLock.Unlock()

	return err
}

func main() {
	lst, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()

	srv := NewChatServer()
	chat.RegisterChatServer(s, srv)

	fmt.Println("New Connection is here!")
	err = s.Serve(lst)

	if err != nil {
		panic(err)
	}
}
