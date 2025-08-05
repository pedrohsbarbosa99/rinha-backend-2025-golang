package main

import (
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"gorinha/internal/processor"
	"time"

	"github.com/fasthttp/router"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

var client *redis.Client

var paymentPending chan models.Payment

func purgePayments(ctx *fasthttp.RequestCtx) {
	client.FlushAll(ctx)
	ctx.SetStatusCode(fasthttp.StatusAccepted)
}

func PostPayments(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusAccepted)
	body := ctx.PostBody()
	go processor.AddToQueue(body)

}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float32 `json:"totalAmount"`
}

func GetSummary(ctx *fasthttp.RequestCtx) {
	summary := map[string]*Summary{
		"default":  {TotalRequests: 0, TotalAmount: 0},
		"fallback": {TotalRequests: 0, TotalAmount: 0},
	}
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	from := int64(0)
	to := time.Date(2400, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339Nano, fromStr)
		if err == nil {
			from = t.UnixNano()
		}
	}

	if toStr != "" {
		t, err := time.Parse(time.RFC3339Nano, toStr)
		if err == nil {
			to = t.UnixNano()
		}
	}

	zs, err := client.ZRangeByScore(
		ctx,
		"payments",
		&redis.ZRangeBy{
			Min: fmt.Sprintf("%d", from),
			Max: fmt.Sprintf("%d", to),
		}).Result()

	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "failed to fetch data"}`)
		return
	}

	for _, val := range zs {
		var p models.Payment
		if err := json.Unmarshal([]byte(val), &p); err != nil {
			continue
		}
		summary[p.Processor].TotalRequests++
		summary[p.Processor].TotalAmount += p.Amount
	}

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
	paymentPending = make(chan models.Payment, 100_000)

	client = redis.NewClient(&redis.Options{
		Addr:           config.REDIS_URL,
		Password:       "",
		DB:             0,
		Protocol:       2,
		MaxActiveConns: 100,
	})
	// go processor.WorkerChecker()
	go processor.WorkerPayments(paymentPending)
	go processor.WorkerDatabase(client, paymentPending)

	r := router.New()
	r.POST("/purge-payments", purgePayments)
	r.POST("/payments", PostPayments)
	r.GET("/payments-summary", GetSummary)

	if err := fasthttp.ListenAndServe(":8080", r.Handler); err != nil {
		panic(err)
	}
}
