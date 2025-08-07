package processor

import (
	"bytes"
	"encoding/binary"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"net/http"
	"time"

	goJson "github.com/goccy/go-json"
)

var queue chan []byte

func AddToQueue(body []byte) {
	queue <- body

}

func WorkerPayments(paymentPending chan models.Payment) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	queue = make(chan []byte, 5_000)

	var data []byte

	for {
		data = <-queue
		var p models.PaymentRequest
		err := goJson.Unmarshal(data, &p)
		if err != nil {
			return
		}
		p.RequestedAt = time.Now().UTC()
		processPayment(httpClient, payment, paymentPending)

	}
}

func WorkerDatabase(db *database.MemClient, paymentPending chan models.Payment) {
	var payment models.Payment
	for {
		payment = <-paymentPending

		var buf bytes.Buffer

		binary.Write(&buf, binary.BigEndian, payment.RequestedAt.UnixNano())
		binary.Write(&buf, binary.BigEndian, payment.Amount)
		data := buf.Bytes()
		db.Put(payment.Processor, data)

	}
}
