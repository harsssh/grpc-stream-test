package handler

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	chatv1 "grpc-stream-test/gen/chat/v1"
	"io"
	"log"
	"sync"
)

type ChatHandler struct {
	clients sync.Map
}

func (h *ChatHandler) receiveLoop(
	stream *connect.BidiStream[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse],
	messageQueue chan *chatv1.Message,
) error {
	for {
		req, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			log.Println("Client disconnected")
			return nil
		}
		if err != nil {
			return err
		}

		msg := req.Message
		messageQueue <- msg
	}

}

func (h *ChatHandler) sendLoop(
	_ *connect.BidiStream[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse],
	messageQueue chan *chatv1.Message,
) error {
	for {
		for msg := range messageQueue {
			if msg.Content != "" {
				h.broadcast(msg)
			}
		}
	}
}

func (h *ChatHandler) ChatStream(ctx context.Context, stream *connect.BidiStream[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse]) error {
	log.Println("New client connected")
	h.clients.Store(stream, struct{}{})
	defer h.clients.Delete(stream)

	messageQueue := make(chan *chatv1.Message)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return h.receiveLoop(stream, messageQueue)
	})
	eg.Go(func() error {
		return h.sendLoop(stream, messageQueue)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (h *ChatHandler) broadcast(msg *chatv1.Message) {
	h.clients.Range(func(k, _ interface{}) bool {
		stream := k.(*connect.BidiStream[chatv1.ChatStreamRequest, chatv1.ChatStreamResponse])
		if err := stream.Send(&chatv1.ChatStreamResponse{Message: msg}); err != nil {
			log.Println("Send error:", err)
			return false
		}
		return true
	})
}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}
