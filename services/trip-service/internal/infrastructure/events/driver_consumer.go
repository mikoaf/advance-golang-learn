package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"

	"github.com/rabbitmq/amqp091-go"
)

type driverConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverConsumer {
	return &driverConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}

func (t *driverConsumer) Listen() error {
	return t.rabbitmq.ConsumeMessage(messaging.DriverTripResponseQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return fmt.Errorf("failed to unmarshal message: %v", err)
		}
		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("failed to unmarshal trip event data: %v", err)
			return fmt.Errorf("failed to unmarshal trip event data: %v", err)
		}
		log.Printf("driver response receive message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := t.handleTripAccepted(ctx, payload.TripID, payload.Driver); err != nil {
				log.Printf("failed to handle trip accept: %v", err)
				return err
			}
		case contracts.DriverCmdTripDecline:
			log.Println("Declined")
			return nil
		}
		log.Printf("unknown trip event: %+v", payload)
		return nil
	})
}

func (t *driverConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pbd.Driver) error {
	// Fetch the first
	trip, err := t.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	if trip == nil {
		return fmt.Errorf("trip was not found %s", tripID)
	}

	// Update the trip
	if err := t.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("failed to update the trip: %v", err)
		return fmt.Errorf("failed to update the trip %s: %v", tripID, err)
	}

	trip, err = t.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	// Driver has been assigned -> publish this event to RabbitMQ
	marshalTrip, err := json.Marshal(trip)
	if err != nil {
		return err
	}

	// Notify the rider that a driver has been assigned
	if err := t.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalTrip,
	}); err != nil {
		return err
	}

	return nil
}
