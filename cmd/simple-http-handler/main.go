package main

import (
	"net/http"

	"github.com/Ghytro/simple-http-multiplexer/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	limiter := handler.NewMuxLimiter(mux)
	mux.HandleFunc("/api/mux", handler.MuxHandler)

	http.ListenAndServe(":8080", limiter)
}
