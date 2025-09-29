package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// TODO: Call trip service
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		// log.Fatal(err)
		log.Printf("error to create trip service client: %v", err)
	}
	defer tripService.Close()

	tripPreview, err := tripService.Client.PreviewTrip(r.Context(), reqBody.toProto())
	if err != nil {
		log.Printf("failed to preview a trip: %v", err)
		http.Error(w, "failed to preview a trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: tripPreview}

	writeJSOn(w, http.StatusCreated, response)
	// io.Copy(w, resp.Body)
	log.Println("SUCCESS")
}

func handleTripStart(w http.ResponseWriter, r *http.Request) {
	var reqBody createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Printf("error to create trip service client: %v", err)
	}
	defer tripService.Close()

	tripStart, err := tripService.Client.CreateTrip(r.Context(), reqBody.toProto())
	if err != nil {
		log.Printf("failed to create a trip: %v", err)
		http.Error(w, "failed to create a trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: tripStart}

	writeJSOn(w, http.StatusCreated, response)
	log.Println("SUCCESS")
}
