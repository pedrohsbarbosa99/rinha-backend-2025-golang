package main

import (
	"fmt"
	"gorinha/internal/models"
	"gorinha/internal/processor"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func PostPayments(c *gin.Context) {
	var p models.PaymentPost

	err := c.ShouldBind(&p)
	if err != nil {
		fmt.Println("PESSIMO PAYLOAD", err.Error())
	}

	go processor.AddToQueue(p)
	if err != nil {
		fmt.Println("Deu RUIM", err.Error())
	}
}

func GetSummary(c *gin.Context) {
	summary := map[string]gin.H{
		"default":  {"totalRequests": 0, "totalAmount": 0.0},
		"fallback": {"totalRequests": 0, "totalAmount": 0.0},
	}

	c.JSON(200, summary)
}

func main() {
	go processor.WorkerChecker()
	for range 2 {
		go processor.Worker()
	}
	router := gin.Default()
	router.POST("/payments", PostPayments)
	router.GET("/payments-summary", GetSummary)
	router.Run()
}
