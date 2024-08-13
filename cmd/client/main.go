package main

import (
	"bufio"
	"connectrpc.com/connect"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
	chatv1 "grpc-stream-test/gen/chat/v1"
	"grpc-stream-test/gen/chat/v1/chatv1connect"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

type ChatClient struct {
	stream   *connect.BidiStreamForClient[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse]
	userName string
}

func NewChatClient(
	userName string,
	stream *connect.BidiStreamForClient[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse],
) *ChatClient {
	stream.Send(&chatv1.ChatStreamRequest{
		Message: &chatv1.Message{
			SenderName: userName,
		},
	})
	return &ChatClient{stream: stream, userName: userName}
}

func (c *ChatClient) send(msg string) error {
	err := c.stream.Send(&chatv1.ChatStreamRequest{
		Message: &chatv1.Message{
			SenderName: c.userName,
			Content:    msg,
		},
	})
	if errors.Is(err, io.EOF) {
		log.Println("Server disconnected")
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

func sendLoop(client *ChatClient) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("> ")
		scanner.Scan()
		msg := scanner.Text()
		if msg == "exit" {
			return nil
		}
		if msg == "" {
			continue
		}

		if err := client.send(msg); err != nil {
			log.Println("Send error:", err)
			continue
		}
	}
}

func receiveLoop(client *ChatClient) error {
	for {
		res, err := client.stream.Receive()
		if errors.Is(err, io.EOF) {
			log.Println("Server disconnected")
			return err
		}
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", res.Message.SenderName, res.Message.Content)
	}
}

func main() {
	c := chatv1connect.NewChatServiceClient(
		&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		},
		"http://localhost:3030",
	)
	ctx := context.Background()
	stream := c.ChatStream(ctx)
	log.Println("Connected to server")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Enter your name: ")
	scanner.Scan()
	name := scanner.Text()

	client := NewChatClient(name, stream)

	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		return sendLoop(client)
	})
	eg.Go(func() error {
		return receiveLoop(client)
	})

	if err := eg.Wait(); err != nil {
		log.Println(err)
	}
}
