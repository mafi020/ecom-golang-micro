package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	"github.com/mafi020/ecom-golang-micro/pkg/events"
)

type orderUseCase interface {
	UpdateStatus(ctx context.Context, orderID int64, status entity.OrderStatus) error
}
type rabbitCoonsumer interface {
	Consume(queue string, h func([]byte) error) error
}

type OrderEventConsumer struct {
	broker       rabbitCoonsumer
	orderUseCase orderUseCase
}

func NewOrderEventConsumer(broker *utils.EventBroker, uc orderUseCase) *OrderEventConsumer {
	return &OrderEventConsumer{
		broker:       broker,
		orderUseCase: uc,
	}
}

// StartListening mounts the background worker handlers to the payment completed stream
func (c *OrderEventConsumer) StartListening() error {
	err := c.broker.Consume("payment.completed", func(body []byte) error {
		// 1. Unmarshal JSON packet back into our typed shared event contract
		var event events.PaymentCompletedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			slog.Error("Dropping malformed event packet data layout", slog.Any("error", err))
			return nil // Return nil so it gets Acked and doesn't poison the queue infinitely
		}

		slog.Info("Asynchronously updating order status to PAID following transaction clearance event",
			slog.Int64("order_id", event.OrderID),
			slog.String("tx_id", event.TransactionID),
		)

		// 2. Invoke our Order UseCase layer to run a local ACID database transaction
		// This moves order status from OrderStatusPending -> OrderStatusPaid
		ctx := context.Background()
		err := c.orderUseCase.UpdateStatus(ctx, event.OrderID, entity.OrderStatusPaid)
		if err != nil {
			return err // Returning an error will Nack the message to preserve eventual consistency
		}

		return nil
	})

	return err
}
