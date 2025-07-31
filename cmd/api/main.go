package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/models"

	"github.com/fasthttp/router"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

var client *redis.Client

func AddToQueue(payment models.PaymentPost, ctx context.Context) {
	data, err := json.Marshal(payment)
	if err != nil {
		return
	}

	client.LPush(ctx, "payment_queue", data).Err()
}

func PostPayments(ctx *fasthttp.RequestCtx) {
	var p models.PaymentPost

	err := json.Unmarshal(ctx.PostBody(), &p)
	if err != nil {
		fmt.Println("PÉSSIMO PAYLOAD", err.Error())
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	go AddToQueue(p, ctx)

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

	resp, err := json.Marshal(summary)
	fmt.Println("SUMMARY: ", summary)
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
	c := config.Config{}
	env := c.LoadEnv()

	client = redis.NewClient(&redis.Options{
		Addr:           env.REDIS_URL,
		Password:       "",
		DB:             0,
		Protocol:       2,
		MaxActiveConns: 100,
	})

	r := router.New()
	r.POST("/payments", PostPayments)
	r.GET("/payments-summary", GetSummary)

	if err := fasthttp.ListenAndServe(":8080", r.Handler); err != nil {
		panic(err)
	}
}
