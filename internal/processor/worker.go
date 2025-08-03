package processor

import (
	"context"
	"gorinha/internal/config"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var env config.Env

var queue chan models.PaymentPost

func AddToQueue(ctx context.Context, payment models.PaymentPost) {
	queue <- payment
}

func WorkerPayments(paymentPending chan models.Payment) {
	httpClient := &http.Client{Timeout: 4 * time.Second}
	queue = make(chan models.PaymentPost, 10_000)
	var wg sync.WaitGroup
	c := config.Config{}
	env = c.LoadEnv()

	const batchSize = 35

	for {
		var payments []models.PaymentPost

		for range batchSize {
			payment := <-queue
			payments = append(payments, payment)
		}

		processPayments(httpClient, payments, &wg, paymentPending)
		wg.Wait()
	}
}

func WorkerDatabase(client *redis.Client, paymentPending chan models.Payment) {
	ctx := context.Background()
	const batchSize = 45
	const flushInterval = 100 * time.Microsecond

	buffer := make([]models.Payment, 0, batchSize)
	timer := time.NewTimer(flushInterval)

	for {

		select {
		case payment := <-paymentPending:
			buffer = append(buffer, payment)

			if len(buffer) >= batchSize {
				database.AddPayments(ctx, client, buffer)
				buffer = buffer[:0]
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(flushInterval)
			}

		case <-timer.C:
			if len(buffer) > 0 {
				database.AddPayments(ctx, client, buffer)
				buffer = buffer[:0]
			}
			timer.Reset(flushInterval)
		}

	}
}
