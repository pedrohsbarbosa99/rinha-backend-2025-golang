package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"gorinha/internal/processor"
	"gorinha/internal/service"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var pendingQueue chan []byte
var db = database.NewStore()

var unixClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", config.OTHER_SOCKET_PATH)
		},
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	},
	Timeout: 3 * time.Second,
}

func GetSummaryInternal(ctx *fasthttp.RequestCtx) {
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	summary, err := service.GetSummary(db, fromStr, toStr)

	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "failed to fetch data"}`)
		return
	}

	resp, err := json.Marshal(summary)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(resp)
}

func GetSummary(ctx *fasthttp.RequestCtx) {
	summaryOther := map[string]*models.Summary{
		"default":  {TotalRequests: 0, TotalAmount: 0},
		"fallback": {TotalRequests: 0, TotalAmount: 0},
	}
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	summary, err := service.GetSummary(db, fromStr, toStr)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "failed to fetch data"}`)
		return
	}

	req, err := http.NewRequest("GET", config.SUMMARY_URL, nil)
	values := req.URL.Query()
	values.Add("from", fromStr)
	values.Add("to", toStr)

	req.URL.RawQuery = values.Encode()

	res, err := unixClient.Do(req)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&summaryOther); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	summary["default"].TotalRequests += summaryOther["default"].TotalRequests
	summary["default"].TotalAmount += summaryOther["default"].TotalAmount

	summary["fallback"].TotalRequests += summaryOther["fallback"].TotalRequests
	summary["fallback"].TotalAmount += summaryOther["fallback"].TotalAmount

	summary["default"].TotalAmount = float32(
		math.Ceil(float64(summary["default"].TotalAmount)*10.0) / 10.0,
	)
	summary["fallback"].TotalAmount = float32(
		math.Ceil(float64(summary["fallback"].TotalAmount)*10.0) / 10.0,
	)

	resp, err := json.Marshal(summary)
	if err != nil {
		fmt.Println(err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(resp)
}

var BodyPool = sync.Pool{

	New: func() any {
		return make([]byte, 0, 100)
	},
}

func handler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/payments":
		ctx.SetStatusCode(fasthttp.StatusAccepted)
		buffer := BodyPool.Get().([]byte)[:0]
		buffer = append(buffer, ctx.PostBody()...)
		pendingQueue <- buffer
	case "/payments-summary":
		GetSummary(ctx)
	case "/internal/payments-summary":
		GetSummaryInternal(ctx)
	}
}

func main() {
	var paymentPool = sync.Pool{
		New: func() any {
			return new(models.PaymentRequest)
		},
	}

	pendingQueue = make(chan []byte, 20_000)
	queue := make(chan *models.PaymentRequest, 20_000)

	os.Remove(config.SOCKET_PATH)

	go processor.AddToQueue(pendingQueue, queue, &paymentPool, &BodyPool)
	go processor.WorkerPayments(db, queue, &paymentPool)

	err := fasthttp.ListenAndServeUNIX(config.SOCKET_PATH, 0777, handler)
	if err != nil {
		panic(err)
	}

}
