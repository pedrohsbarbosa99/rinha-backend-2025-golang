package processor

import (
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
	"sync"
	"time"

	goJson "github.com/goccy/go-json"
)

func AddToQueue(pendingQueue chan []byte, queue chan *models.PaymentRequest, paymentPool *sync.Pool) {
	for {
		body := <-pendingQueue
		p := paymentPool.Get().(*models.PaymentRequest)

		p.CorrelationId = ""
		p.Amount = 0
		p.Err = false

		goJson.Unmarshal(body, p)
		p.RequestedAt = time.Now().UTC()

		queue <- p

	}
}

func WorkerPayments(db *database.Store, queue chan *models.PaymentRequest, paymentPool *sync.Pool) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	for {
		payment := <-queue
		processor, err := processPayment(httpClient, payment)
		if err != nil {
			payment.Err = true
			queue <- payment
			continue
		}
		db.Put(processor, *payment)
		paymentPool.Put(payment)

	}
}
