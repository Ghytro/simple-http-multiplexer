package main

import (
	"net/http"

	"github.com/Ghytro/simple-http-multiplexer/internal/handler"
	"github.com/Ghytro/simple-http-multiplexer/internal/server"
)

func main() {
	mux := http.NewServeMux()
	limiter := handler.NewMuxLimiter(mux)
	mux.HandleFunc("/api/mux", handler.MuxHandler)

	server.NewGracefulShutdownServer(":8080", limiter).Listen()
}
