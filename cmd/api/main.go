package main

import (
	"encoding/json"
	"fmt"
	"gorinha/internal/database"
	"gorinha/internal/processor"
	"gorinha/internal/service"
	"sync"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var db *database.MemClient

var bodyQueue chan []byte

var BodyPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 512)
	},
}

func PostPayments(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusAccepted)
	buffer := BodyPool.Get().([]byte)[:0]
	buffer = append(buffer, ctx.PostBody()...)
	bodyQueue <- buffer

}

func GetSummary(ctx *fasthttp.RequestCtx) {
	time.Sleep(100 * time.Millisecond)
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	summary, err := service.GetSummary(db, fromStr, toStr)

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

func main() {
	bodyQueue = make(chan []byte, 100_000)

	db = database.NewMemClient()

	go processor.AddToQueue(db, bodyQueue, &BodyPool)

	r := router.New()
	r.POST("/payments", PostPayments)
	r.GET("/payments-summary", GetSummary)

	if err := fasthttp.ListenAndServe(":8080", r.Handler); err != nil {
		panic(err)
	}
}
