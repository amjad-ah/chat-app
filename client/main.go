package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/amjad-ah/chat-app/chat"
	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) != 3 {
		panic("Missing data")
	}

	conn, err := grpc.Dial(os.Args[1], grpc.WithInsecure())

	handelError(err)

	defer conn.Close()

	c := chat.NewChatClient(conn)

	ctx := context.Background()

	stream, err := c.Chat(ctx)

	handelError(err)

	waitc := make(chan struct{})

	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}

			handelError(err)

			fmt.Println(msg.User + " said: " + msg.Message)
		}
	}()

	fmt.Println("New connection started!")

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		msg := scanner.Text()
		if msg == "bye" {
			err := stream.CloseSend()
			handelError(err)
			break
		}

		err := stream.Send(&chat.ChatMessage{
			User:    os.Args[2],
			Message: msg,
		})

		handelError(err)
	}

	<-waitc
}

func handelError(err error) {
	if err != nil {
		panic(err)
	}
}
