package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ghytro/simple-http-multiplexer/internal/config"
)

type GracefulShutdownedServer struct {
	serv http.Server
}

func NewGracefulShutdownServer(addr string, handler http.Handler) *GracefulShutdownedServer {
	return &GracefulShutdownedServer{
		serv: http.Server{
			Addr:        addr,
			Handler:     handler,
			ReadTimeout: time.Second * config.RequestHandleTimeout,
		},
	}
}

func (s *GracefulShutdownedServer) Listen() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := s.serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("Server is listenning at %s", s.serv.Addr)
	<-done
	log.Println("Gracefully shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := s.serv.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %+v", err)
	}
	log.Println("Gracefully shutted down")
}
