package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/database"
	"gorinha/internal/processor"
	"gorinha/internal/service"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var pendingQueue chan []byte
var db = database.NewStore()

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

var BodyPool = sync.Pool{

	New: func() any {
		return make([]byte, 0, 100)
	},
}

func handler(ctx *fasthttp.RequestCtx) {
	switch {
	case bytes.Equal(ctx.Path(), []byte("/payments")):
		ctx.SetStatusCode(fasthttp.StatusAccepted)
		buffer := BodyPool.Get().([]byte)[:0]
		buffer = append(buffer, ctx.PostBody()...)
		pendingQueue <- buffer
	case bytes.Equal(ctx.Path(), []byte("/payments-summary")):
		GetSummary(ctx)
	case bytes.Equal(ctx.Path(), []byte("/internal/payments-summary")):
		GetSummaryInternal(ctx)
	}
}

func main() {
	db := database.NewClient()
	pendingQueue = make(chan []byte, 20_000)

	os.Remove(config.SOCKET_PATH)

	go processor.AddToQueue(pendingQueue, &BodyPool, db)

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
