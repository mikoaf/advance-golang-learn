package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Remove imports of internal packages
	"ride-sharing/shared/env"
)

var (
	// httpAddr = env.GetString("HTTP_ADDR", ":8081")
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")

	mux := http.NewServeMux()

	mux.HandleFunc("POST /trip/preview", handleTripPreview)
	mux.HandleFunc("/ws/drivers", handleDriversWebsocket)
	mux.HandleFunc("/ws/riders", handleRidersWebsocket)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("error starting the server: %v", err)
	case sig := <-shutdown:
		log.Printf("server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("could not stop the server gracefully: %v", err)
			server.Close()
		}

	}

	// if err := server.ListenAndServe(); err != nil {
	// 	log.Fatalf("HTTP server error: %v", err)
	// }
}
