package database

import (
	"context"
	"encoding/json"
	"fmt"
	"gorinha/internal/models"

	"github.com/redis/go-redis/v9"
)

func AddPayments(
	ctx context.Context,
	client *redis.Client,
	payments []models.Payment,
) error {
	key := "payments"

	zs := make([]redis.Z, 0, len(payments))

	for _, p := range payments {
		data, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal payment: %w", err)
		}

		zs = append(zs, redis.Z{
			Score:  float64(p.RequestedAt.UnixNano()),
			Member: data,
		})
	}

	if err := client.ZAdd(ctx, key, zs...).Err(); err != nil {
		return fmt.Errorf("failed to add payments: %w", err)
	}

	return nil
}
