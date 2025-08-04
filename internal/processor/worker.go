package processor

import (
	"context"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
	"sync"
	"time"

	goJson "github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

var queue chan models.PaymentRequest

func AddToQueue(body []byte) {
	var p models.PaymentRequest
	err := goJson.Unmarshal(body, &p)
	if err != nil {
		return
	}
	p.RequestedAt = time.Now().UTC()
	queue <- p
}

func WorkerPayments(paymentPending chan models.Payment) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	queue = make(chan models.PaymentRequest, 10_000)

	var wg sync.WaitGroup

	const batchSize = 10

	for {
		var payments []models.PaymentRequest

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
	const flushInterval = 150 * time.Microsecond

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
