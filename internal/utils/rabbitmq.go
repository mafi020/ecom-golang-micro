package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewEventBroker connects to RabbitMQ and prepares a persistent communication channel
func NewEventBroker(amqpURL string) (*EventBroker, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		slog.Info("failed to connect to rabbitmq", slog.Any("err", err))
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		slog.Info("failed to open channel", slog.Any("err", err))
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &EventBroker{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish serializes any payload to JSON and dispatches it onto a named RabbitMQ exchange or queue
func (b *EventBroker) Publish(ctx context.Context, queueName string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Info("failed to marshal event payload", slog.Any("err", err))
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Declare the queue to ensure it exists before publishing
	_, err = b.channel.QueueDeclare(
		queueName, // name
		true,      // durable (survives broker crashes)
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		slog.Info("failed to declare queue", slog.Any("err", err))
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Stream the data packet over the wire network
	err = b.channel.PublishWithContext(ctx,
		"",        // exchange (empty string uses the default direct exchange)
		queueName, // routing key (maps directly to queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Saves message to disk so it isn't lost if RabbitMQ restarts
			Body:         body,
		},
	)

	if err != nil {
		slog.Info("failed to publish message", slog.Any("err", err))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Consume registers a persistent background listener loop on a specific queue
func (b *EventBroker) Consume(queueName string, handler func(body []byte) error) error {
	// 1. Guarantee the targeted queue exists on the broker

	_, err := b.channel.QueueDeclare(
		queueName, // name
		true,      // durable (survives broker crashes)
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to declare consumer queue: %w", err)
	}

	// 2. Set QoS (Quality of Service) prefetch count to 1.
	// This ensures RabbitMQ only gives the worker 1 message at a time, protecting memory.
	err = b.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to configure worker qos: %w", err)
	}

	// 3. Register the consumer stream channel channel loop
	msgs, err := b.channel.Consume(
		queueName,
		"",    // consumer tag identifier
		false, // auto-ack turned OFF (we handle ack manually after database updates succeed)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer channel: %w", err)
	}

	// 4. Launch the background execution daemon processor loop
	go func() {
		slog.Info("RabbitMQ worker daemon successfully attached to queue", slog.String("queue", queueName))
		for d := range msgs {
			// Process the raw byte array via our domain business callback function
			err := handler(d.Body)
			if err != nil {
				slog.Error("Worker failed to process event payload; rejecting message", slog.Any("error", err))
				// Nack tells RabbitMQ to put the message back in the queue so it can be retried later (requeue = true)
				_ = d.Nack(false, true)
				continue
			}

			// 🚀 CRITICAL: Tell RabbitMQ the database operation succeeded. Safe to delete!
			_ = d.Ack(false)
		}
	}()

	return nil
}

// Close safely terminates channels and network socket connections
func (b *EventBroker) Close() {
	if b.channel != nil {
		b.channel.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
}
