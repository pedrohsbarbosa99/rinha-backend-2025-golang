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

var requestQueue chan []byte
var queue chan models.PaymentRequest

func AddToQueue(body []byte) {
	requestQueue <- body
}

func WorkerJsonParser() {
	requestQueue = make(chan []byte, 5_000)
	for {
		body := <-requestQueue
		var p models.PaymentRequest
		err := goJson.Unmarshal(body, &p)
		if err != nil {
			return
		}
		p.RequestedAt = time.Now().UTC()
		queue <- p
	}
}

func WorkerPayments(paymentPending chan models.Payment) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	queue = make(chan models.PaymentRequest, 5_000)

	var payment models.PaymentRequest

	for {
		payment = <-queue
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
