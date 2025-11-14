package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"ride-sharing/services/payment-service/internal/events"
	"ride-sharing/services/payment-service/internal/infrastructure/stripe"
	"ride-sharing/services/payment-service/internal/service"
	"ride-sharing/services/payment-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9004")

func main() {
	// Initialize tracing
	tracerCfg := tracing.Config{
		ServiceName:    "payment-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	rabbitmqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")

	stripeCfg := types.PaymentConfig{
		StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
		SuccessURL:      env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"),
		CancelURL:       env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"),
	}

	if stripeCfg.StripeSecretKey == "" {
		log.Fatalf("STRIP_SECRET_KEY is not set")
		return
	}

	// Stripe processor
	paymentProcessor := stripe.NewStripeClient(&stripeCfg)

	// Service
	svc := service.NewPaymentService(paymentProcessor)

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitmqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Trip consumer
	tripConsumer := events.NewTripConsumer(rabbitmq, svc)
	go tripConsumer.Listen()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down payment service")
}
