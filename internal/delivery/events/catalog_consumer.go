package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mafi020/ecom-golang-micro/pkg/events"
)

type catalogUseCase interface {
	// Assuming your usecase (or product repo) has a method to update stock directly
	UpdateProductStock(ctx context.Context, productID int64, newStock int32) error
}

type CatalogEventConsumer struct {
	broker         rabbitConsumer
	catalogUseCase catalogUseCase
}

func NewCatalogEventConsumer(broker rabbitConsumer, uc catalogUseCase) *CatalogEventConsumer {
	return &CatalogEventConsumer{
		broker:         broker,
		catalogUseCase: uc,
	}
}

// StartListening mounts the background worker onto the order.placed RabbitMQ queue
func (c *CatalogEventConsumer) StartListening() error {
	err := c.broker.Consume("order.placed", func(body []byte) error {
		var event events.OrderPlacedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			slog.Error("[CATALOG-CONSUMER] Dropping malformed order event payload", slog.Any("error", err))
			return nil // Return nil so it gets Acked and cleared from the queue
		}

		slog.Info("[CATALOG-CONSUMER] Processing stock adjustments for placed order", slog.Int64("order_id", event.OrderID))

		ctx := context.Background()

		// Loop through all product updates packaged inside the event message
		for productID, newStock := range event.StockUpdates {
			slog.Info("[CATALOG-CONSUMER] Asynchronously updating product stock allocation",
				slog.Any("product_id", productID),
				slog.Any("new_stock", newStock),
			)

			// Execute local usecase logic to run a single database transaction update statement
			if err := c.catalogUseCase.UpdateProductStock(ctx, productID, newStock); err != nil {
				slog.Error("[CATALOG-CONSUMER] Failed to process database inventory update",
					slog.Int64("product_id", productID),
					slog.Any("error", err),
				)
				return err // Nacks the message to preserve event safety and retry later
			}
		}

		return nil
	})

	return err
}
