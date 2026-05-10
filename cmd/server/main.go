package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/anivaryam/proxy-relay/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if os.Getenv("PROXY_AUTH_TOKEN") == "" {
		log.Fatal("PROXY_AUTH_TOKEN environment variable is required")
	}

	auth := server.NewAuth()
	srv := server.New(":"+port, auth)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down")
	srv.Close()
}
