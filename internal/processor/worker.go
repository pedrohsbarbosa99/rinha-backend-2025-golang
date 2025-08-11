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

		*p = models.PaymentRequest{}

		goJson.Unmarshal(body, p)
		p.RequestedAt = time.Now().UTC()

		queue <- p

	}
}

func WorkerPayments(db *database.Store, queue chan *models.PaymentRequest, paymentPool *sync.Pool) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	for {
		payment := <-queue
		processor, err := processPayment(httpClient, *payment)
		if err != nil {
			queue <- &models.PaymentRequest{
				CorrelationId: payment.CorrelationId,
				Amount:        payment.Amount,
				RequestedAt:   payment.RequestedAt,
				Err:           true,
			}
			paymentPool.Put(payment)
			time.Sleep(time.Second)
			continue
		}
		db.Put(processor, *payment)
		paymentPool.Put(payment)

	}
}
