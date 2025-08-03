package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthReturn struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

func GetHealth(url string) (h HealthReturn) {
	fullURL := fmt.Sprintf("%s/payments/service-health", url)
	fmt.Printf("[LOG] [GetHealth] Verificando health de: %s\n", fullURL)

	res, err := http.Get(fullURL)
	if err != nil {
		fmt.Printf("[LOG] [GetHealth] Erro ao fazer request para %s: %v\n", fullURL, err)
		return
	}
	fmt.Println("STATUS: ", res.StatusCode)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("[LOG] [GetHealth] Erro ao ler body de %s: %v\n", fullURL, err)
		return
	}

	err = json.Unmarshal(body, &h)
	if err != nil {
		fmt.Printf("[LOG] [GetHealth] Erro ao fazer unmarshal de %s: %v, body: %s\n", fullURL, err, string(body))
		return
	}

	fmt.Printf("[LOG] [GetHealth] Health de %s: failing=%v, minResponseTime=%d\n", url, h.Failing, h.MinResponseTime)
	return
}

func ChoiceProcessor() (url string, processor string, fail bool) {
	healthDefault := GetHealth("http://payment-processor-default:8080")
	url = "http://payment-processor-default:8080"
	processor = "default"
	fail = healthDefault.Failing

	if !healthDefault.Failing {
		if healthDefault.MinResponseTime < 130 {
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
			fail = healthFallback.Failing
			return
		}
	}

	fail = true
	return
}

func WorkerChecker(client *redis.Client) {
	c := config.Config{}
	env := c.LoadEnv()
	ctx := context.Background()

	initialValue := fmt.Sprintf("default##%s##false", env.PROCESSOR_DEFAULT_URL)
	client.Set(ctx, "processor", initialValue, 10*time.Second).Err()

	for {
		url, processor, fail := ChoiceProcessor()
		fmt.Printf("[LOG] [WorkerChecker] SELECIONANDO: %s -> %s\n", processor, url)

		newValue := fmt.Sprintf("%s##%s##%v", processor, url, fail)
		client.Set(ctx, "processor", newValue, 10*time.Second).Err()

	}
}
