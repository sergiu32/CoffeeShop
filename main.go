package main

import (
	"CoffeeShop/shopapi"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func createChannel() (chan os.Signal, func()) {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	return stopCh, func() {
		close(stopCh)
	}
}

func main() {
	shopapi.InitDefaultConfig()
	shopapi.InitDb("Data")

	mux := http.NewServeMux()

	s := &http.Server{Addr: ":8080", Handler: mux}
	go shopapi.StartHttpServer(s, mux)

	stopCh, closeCh := createChannel()
	defer closeCh()
	log.Println("notified:", <-stopCh)

	shopapi.ShutdownHttpServer(context.Background(), s)
}
