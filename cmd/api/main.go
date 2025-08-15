package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"gorinha/internal/processor"
	"gorinha/internal/service"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var pendingQueue chan []byte
var db = database.NewStore()

var httpClient = &http.Client{
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
	time.Sleep(50 * time.Millisecond)
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

	res, err := httpClient.Do(req)
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
	switch {
	case bytes.Equal(ctx.Path(), []byte("/payments")):
		go func() {
			buffer := BodyPool.Get().([]byte)[:0]
			buffer = append(buffer, ctx.PostBody()...)
			pendingQueue <- buffer
		}()
		ctx.SetStatusCode(fasthttp.StatusAccepted)
	case bytes.Equal(ctx.Path(), []byte("/payments-summary")):
		GetSummary(ctx)
	case bytes.Equal(ctx.Path(), []byte("/internal/payments-summary")):
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

	srv := &fasthttp.Server{
		Handler:                       handler,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
	}

	err := srv.ListenAndServe(":8080")
	if err != nil {
		panic(err)
	}

}
