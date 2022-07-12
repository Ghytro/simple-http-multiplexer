package main

import (
	"net/http"
	"strconv"

	"github.com/Ghytro/simple-http-multiplexer/internal/config"
	"github.com/Ghytro/simple-http-multiplexer/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	limiter := handler.NewMuxLimiter(mux)
	mux.HandleFunc("/api/mux", handler.MuxHandler)

	http.ListenAndServe(":"+strconv.Itoa(config.Port), limiter)
}
