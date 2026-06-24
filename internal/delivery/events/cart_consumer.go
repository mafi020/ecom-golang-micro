package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mafi020/ecom-golang-micro/pkg/events"
)

type cartUseCase interface {
	ClearCart(ctx context.Context, userID int64) error
}

type rabbitConsumer interface {
	Consume(queue string, h func([]byte) error) error
}

type CartEventConsumer struct {
	broker      rabbitConsumer
	cartUseCase cartUseCase
}

func NewCartEventConsumer(broker rabbitConsumer, uc cartUseCase) *CartEventConsumer {
	return &CartEventConsumer{
		broker:      broker,
		cartUseCase: uc,
	}
}

// StartListening registers a background worker onto the same payment completed queue
func (c *CartEventConsumer) StartListening() error {
	// 🚀 MULTI-CONSUMER PATTERN: Both order-service and cart-service consume from this event stream
	err := c.broker.Consume("payment.completed", func(body []byte) error {
		var event events.PaymentCompletedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			slog.Error("[CART-CONSUMER] Dropping malformed event packet data layout", slog.Any("error", err))
			return nil
		}

		// Ensure we actually have a UserID to clear the correct cart
		if event.UserID == 0 {
			slog.Warn("[CART-CONSUMER] Event missing UserID; skipping cart clearance", slog.Int64("order_id", event.OrderID))
			return nil
		}

		slog.Info("[CART-CONSUMER] Asynchronously clearing user shopping cart following successful checkout payment",
			slog.Int64("user_id", event.UserID),
			slog.Int64("order_id", event.OrderID),
		)

		// Invoke your local Cart UseCase business logic to truncate/delete cart item database rows
		ctx := context.Background()
		if err := c.cartUseCase.ClearCart(ctx, event.UserID); err != nil {
			slog.Error("[CART-CONSUMER] Failed to clear user database cart rows", slog.Any("error", err))
			return err
		}

		return nil
	})

	return err
}
