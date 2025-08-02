package processor

import (
	"context"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"sync"

	"github.com/redis/go-redis/v9"
)

var env config.Env

func WorkerPayments(client *redis.Client, q <-chan models.PaymentPost) {
	c := config.Config{}
	env = c.LoadEnv()
	ctx := context.Background()

	const batchSize = 30

	for {
		var payments []models.PaymentPost

		for range batchSize {
			payment := <-q
			payments = append(payments, payment)
		}

		var wg sync.WaitGroup
		processPayments(ctx, client, payments, &wg)
		wg.Wait()
	}
}
