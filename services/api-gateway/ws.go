package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"
)

// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebsocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Print("no user ID provided")
		return
	}

	// Add connection to manager
	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

	// Initialize queue consumers
	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
	}

	for _, queue := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, queue)

		if err := consumer.Start(); err != nil {
			log.Printf("failed to start consumer for queue: %s | error: %v", queue, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}

func handleDriversWebsocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Print("no user ID provided")
		return
	}

	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Print("no package slug provided")
		return
	}

	// Add connection to manager
	connManager.Add(userID, conn)

	ctx := r.Context()

	driverService, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Closing connections
	defer func() {
		connManager.Remove(userID)
		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})
		driverService.Close()
		log.Println("Driver unregistered: ", userID)
	}()

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})

	if err != nil {
		log.Printf("error registering driver: %v", err)
		return
	}

	// msg := contracts.WSMessage{
	// 	Type: "driver.cmd.register",
	// 	Data: driverData.Driver,
	// }

	// if err := conn.WriteJSON(msg); err != nil {
	// 	log.Printf("error sending message: %v", err)
	// 	return
	// }

	if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}); err != nil {
		log.Printf("error sending message: %v", err)
		return
	}

	// Initialize queue consumers
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, queue := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, queue)

		if err := consumer.Start(); err != nil {
			log.Printf("failed to start consumer for queue: %s | error: %v", queue, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("failed to unmarshal driver message: %v", err)
			continue
		}

		// Handle the different message type
		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			continue
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			if err := rb.PublishMessage(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("failed to publishing driver message to RabbitMQ: %v", err)
				continue
			}
		default:
			log.Printf("unknown message type: %s", driverMsg.Type)
		}

		log.Printf("Received message: %s", message)
	}

}
