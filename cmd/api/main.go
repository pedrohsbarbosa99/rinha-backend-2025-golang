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
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var pendingQueue chan []byte
var db = database.NewStore()

func PostPayments(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusAccepted)
	body := ctx.PostBody()
	pendingQueue <- body

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

func newUnixSocketClient() *http.Client {
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.Dial("unix", config.OTHER_SOCKET_PATH)
	}

	transport := &http.Transport{
		DialContext: dialer,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
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

	client := newUnixSocketClient()
	req, err := http.NewRequest("GET", config.SUMMARY_URL, nil)
	values := req.URL.Query()
	values.Add("from", fromStr)
	values.Add("to", toStr)

	req.URL.RawQuery = values.Encode()

	res, err := client.Do(req)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "internal error"}`)
		return
	}
	if err := json.Unmarshal(body, &summaryOther); err != nil {
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

func handler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/payments":
		PostPayments(ctx)
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

	go processor.AddToQueue(pendingQueue, queue, &paymentPool)
	go processor.WorkerPayments(db, queue, &paymentPool)

	err := fasthttp.ListenAndServeUNIX(config.SOCKET_PATH, 0777, handler)
	if err != nil {
		panic(err)
	}

}
