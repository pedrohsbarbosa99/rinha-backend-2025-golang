package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"gorinha/internal/processor"

	"github.com/fasthttp/router"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

var client *redis.Client
var queue chan models.PaymentPost

func AddToQueue(ctx context.Context, body []byte, q <-chan models.PaymentPost) {
	var payment models.PaymentPost
	json.Unmarshal(body, &payment)
	queue <- payment
}

func PostPayments(ctx *fasthttp.RequestCtx) {
	body := ctx.PostBody()

	go AddToQueue(ctx, body, queue)

	ctx.SetStatusCode(fasthttp.StatusAccepted)
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func GetSummary(ctx *fasthttp.RequestCtx) {
	summary := map[string]*Summary{
		"default":  {TotalRequests: 0, TotalAmount: 0},
		"fallback": {TotalRequests: 0, TotalAmount: 0},
	}

	// from := string(ctx.QueryArgs().Peek("from"))
	// to := string(ctx.QueryArgs().Peek("to"))

	defaults, err := client.ZRangeByScore(ctx, "payments:default", &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	fallbacks, err := client.ZRangeByScore(ctx, "payments:fallback", &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	for range defaults {
		summary["default"].TotalRequests++
		summary["default"].TotalAmount += 19.90
	}
	for range fallbacks {
		summary["fallback"].TotalRequests++
		summary["fallback"].TotalAmount += 19.90
	}
	fmt.Println("SUMMARY: ", len(fallbacks), len(defaults))

	resp, err := json.Marshal(summary)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(resp)
}

func main() {
	queue = make(chan models.PaymentPost, 10000)
	c := config.Config{}
	env := c.LoadEnv()

	client = redis.NewClient(&redis.Options{
		Addr:           env.REDIS_URL,
		Password:       "",
		DB:             0,
		Protocol:       2,
		MaxActiveConns: 50,
	})
	go processor.WorkerPayments(client, queue)

	r := router.New()
	r.POST("/payments", PostPayments)
	r.GET("/payments-summary", GetSummary)

	if err := fasthttp.ListenAndServe(":8080", r.Handler); err != nil {
		panic(err)
	}
}
