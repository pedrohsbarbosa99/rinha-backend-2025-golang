package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func AddPayment(
	ctx context.Context,
	client *redis.Client,
	cid string,
	processor string,
	amount float32,
	requestedAt time.Time,
) {
	client.ZAdd(ctx, fmt.Sprintf(`payments:%s`, processor), redis.Z{
		Score:  float64(requestedAt.Unix()),
		Member: cid,
	})

}
