package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthReturn struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

func GetHealth(url string) (h HealthReturn) {
	res, err := http.Get(fmt.Sprintf("%s/payments/service-health", url))
	if err != nil {
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return
	}

	err = json.Unmarshal(body, &h)

	return
}

func ChoiceProcessor() (url string, processor string) {
	healthDefault := GetHealth("http://payment-processor-default:8080")
	if !healthDefault.Failing {
		if healthDefault.MinResponseTime < 130 {
			url = "http://payment-processor-default:8080"
			processor = "default"
			return
		}
	}

	healthFallback := GetHealth("http://payment-processor-fallback:8080")
	if !healthFallback.Failing {
		if !healthDefault.Failing &&
			(float32(healthDefault.MinResponseTime) >
				float32(healthFallback.MinResponseTime)*1.2) {
			url = "http://payment-processor-fallback:8080"
			processor = "fallback"
		}
		return
	}
	return
}

func WorkerChecker() {
	c := Config{}
	env := c.LoadEnv()
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     env.REDIS_URL,
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	fmt.Println("Redis conectado:", pong)
	client.Set(ctx, "processor",
		fmt.Sprintf(
			"%s:%s",
			"default",
			env.PROCESSOR_DEFAULT_URL,
		),
		5*time.Second,
	)

	for {

		url, processor := ChoiceProcessor()
		client.Set(ctx,
			"processor",
			fmt.Sprintf("%s:%s", processor, url),
			5*time.Second,
		)
		time.Sleep(5 * time.Second)
	}
}
