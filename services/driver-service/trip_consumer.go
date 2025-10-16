package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {
	return &tripConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}

func (t *tripConsumer) Listen() error {
	return t.rabbitmq.ConsumeMessage(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return fmt.Errorf("failed to unmarshal message: %v", err)
		}
		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("failed to unmarshal trip event data: %v", err)
			return fmt.Errorf("failed to unmarshal trip event data: %v", err)
		}
		log.Printf("driver receive message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return t.handleFindAndNotifyDrivers(ctx, payload)
		}
		log.Printf("unknown trip event: %+v", payload)
		return nil
	})
}

func (t *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableIDs := t.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)

	log.Printf("Found suitable drivers %d", len(suitableIDs))

	if len(suitableIDs) == 0 {
		// Notify the rider that no drivers are available
		if err := t.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("failed to publish message to exchange: %v", err)
			return fmt.Errorf("failed to publish message to exchange: %v", err)
		}
		return nil
	}

	// Get a random index from the matching drivers
	randomIndex := rand.Intn(len(suitableIDs))

	suitableDriverID := suitableIDs[randomIndex]
	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Notify the driver about a potential trip
	if err := t.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEvent,
	}); err != nil {
		log.Printf("failed to publish message to exchange: %v", err)
		return fmt.Errorf("failed to publish message to exchange: %v", err)
	}

	return nil
}
