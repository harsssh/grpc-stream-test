package main

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"grpc-stream-test/gen/chat/v1/chatv1connect"
	"grpc-stream-test/internal/handler"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	chatHandler := handler.NewChatHandler()
	path, h := chatv1connect.NewChatServiceHandler(chatHandler)
	mux.Handle(path, h)
	http.ListenAndServe(":3030", h2c.NewHandler(mux, &http2.Server{}))
}
