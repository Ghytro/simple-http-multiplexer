package handler

import (
	"fmt"
	"net/http"

	"github.com/Ghytro/simple-http-multiplexer/internal/config"
	"github.com/Ghytro/simple-http-multiplexer/internal/limiter"
)

type MuxLimiter struct {
	connLimiter *limiter.Limiter
	errHandler  *MuxErrorHandler
}

func NewMuxLimiter(mux http.Handler) *MuxLimiter {
	return &MuxLimiter{
		connLimiter: limiter.NewLimiter(config.MaxIncomingConnections),
		errHandler:  NewMuxErrorHandler(mux),
	}
}

func (ml *MuxLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ml.connLimiter.ConnAllowed() {
		ml.errHandler.ServeHTTP(w, r)
		ml.connLimiter.Disconnected()
		return
	}
	w.Header().Add("Retry-After", fmt.Sprint(int(config.RequestHandleTimeout)/2))
	w.WriteHeader(http.StatusTooManyRequests)
}

type MuxErrorHandler struct {
	mux http.Handler
}

func NewMuxErrorHandler(mux http.Handler) *MuxErrorHandler {
	return &MuxErrorHandler{mux: mux}
}

func (eh *MuxErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// the server accepts only post requests, so this error handler can be global
	if r.Method != "POST" {
		http.Error(
			w,
			"expected POST HTTP method, but got: "+r.Method,
			http.StatusBadRequest,
		)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	eh.mux.ServeHTTP(w, r)
}
