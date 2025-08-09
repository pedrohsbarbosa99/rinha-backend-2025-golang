package processor

import (
	"bytes"
	"encoding/binary"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
	"sync"
	"time"

	goJson "github.com/goccy/go-json"
)

var queue = make(chan *models.PaymentRequest, 5_000)

var paymentRequestPool = sync.Pool{
	New: func() any { return &models.PaymentRequest{} },
}

func AddToQueue(paymentRequestQueue chan []byte) {
	for body := range paymentRequestQueue {
		p := paymentRequestPool.Get().(*models.PaymentRequest)
		err := goJson.Unmarshal(body, p)
		if err != nil {
			return
		}
		p.RequestedAt = time.Now().UTC()

		queue <- p
	}
}

func WorkerPayments(paymentPending chan models.Payment) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	for payment := range queue {
		err := processPayment(httpClient, *payment, paymentPending)
		if err != nil {
			time.Sleep(time.Second)
		}
		paymentRequestPool.Put(payment)

	}
}

var bufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 12))
	},
}

var paymentPendingPool = sync.Pool{
	New: func() any { return &models.Payment{} },
}

func WorkerDatabase(db *database.MemClient, paymentPending chan models.Payment) {
	for {
		payment := paymentPendingPool.Get().(*models.Payment)
		*payment = <-paymentPending

		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()

		binary.Write(buf, binary.BigEndian, payment.RequestedAt.UnixNano())
		binary.Write(buf, binary.BigEndian, payment.Amount)
		data := buf.Bytes()
		db.Put(payment.Processor, data)
		bufPool.Put(buf)
		paymentPendingPool.Put(payment)

	}
}
