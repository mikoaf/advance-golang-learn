package main

import (
	"log"
	"net/http"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
)

func main() {

	mux := http.NewServeMux()
	// ctx := context.Background()
	// fare := &domain.RideFareModel{
	// 	UserID: "42",
	// }
	inMemRepo := repository.NewInMemRepository()
	svc := service.NewService(inMemRepo)
	httpHandler := h.HttpHandler{Service: svc}
	mux.HandleFunc("POST /preview", httpHandler.HandlePreview)
	// t, err := svc.CreateTrip(ctx, fare)
	// if err != nil {
	// 	log.Println(err)
	// }

	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
	// for {
	// 	time.Sleep(1 * time.Second)
	// }
}
