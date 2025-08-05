package processor

import (
	"context"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
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

	for {
		if processorHealth.Failing {
			time.Sleep(2 * time.Second)
		}

		payment := <-queue
		err := processPayment(httpClient, payment, paymentPending)
		if err != nil {
			time.Sleep(time.Second)
		}
	}
}

func WorkerDatabase(client *redis.Client, paymentPending chan models.Payment) {
	ctx := context.Background()
	const batchSize = 500
	const flushInterval = 200 * time.Microsecond

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
