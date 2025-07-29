package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var queue chan PaymentPost

func AddToQueue(payment PaymentPost) {
	queue <- payment
}

var client *redis.Client

func getProcessor(ctx context.Context) (processor string, url string) {
	data, err := client.Get(ctx, "processor").Result()
	if err == redis.Nil {
		dt := strings.Split(data, ":")
		fmt.Println("dt", dt)
		processor = dt[0]
		url = dt[1]
	}
	return

}

func PostPayment(payment PaymentPost, url string) (
	requestedAt time.Time,
	err error,
) {
	requestedAt = time.Now().UTC()
	data := map[string]any{
		"correlationId": payment.CorrelationId,
		"amount":        payment.Amount,
		"requestedAt":   requestedAt,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	res, err := http.Post(
		fmt.Sprintf("%s/payments", url),
		"application/json",
		bytes.NewBuffer(payload),
	)

	if err != nil {
		fmt.Println("CHEGASTES", err.Error())
		return
	}

	if res.StatusCode != 200 {
		return requestedAt, errors.New("Deu erro na request")
	}

	return
}

func Worker() {
	c := Config{}
	env := c.LoadEnv()
	ctx := context.Background()
	client = redis.NewClient(&redis.Options{
		Addr:     env.REDIS_URL,
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	queue = make(chan PaymentPost, 20000)

	for {
		for payment := range queue {
			_, url := getProcessor(ctx)
			requestedAt, err := PostPayment(payment, url)
			if err == nil {
				fmt.Println(requestedAt)
			}
			time.Sleep(time.Duration(1))
		}
	}
}
