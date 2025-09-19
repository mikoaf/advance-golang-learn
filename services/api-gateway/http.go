package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "failed to parse request body: %v", http.StatusInternalServerError)
		return
	}

	// url := fmt.Sprintf("http://trip-service:8083/preview?userID=%s", reqBody.UserID)
	url := "http://trip-service:8083/preview"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		http.Error(w, "failed to request", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("trip-service error: %s", body), resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)

	var trip any
	if err := json.Unmarshal(body, &trip); err != nil {
		http.Error(w, "failed to decode trip-service response", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSOn(w, http.StatusCreated, response)
	// io.Copy(w, resp.Body)
	log.Println("SUCCESS")
}
