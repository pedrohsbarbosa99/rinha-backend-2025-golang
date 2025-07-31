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

func ChoiceProcessor() (url string, processor string) {
	healthDefault := GetHealth("http://payment-processor-default:8080")
	fmt.Printf("[LOG] [ChoiceProcessor] Health Default: failing=%v, minResponseTime=%d\n", healthDefault.Failing, healthDefault.MinResponseTime)
	
	if !healthDefault.Failing {
		if healthDefault.MinResponseTime < 130 {
			url = "http://payment-processor-default:8080"
			processor = "default"
			fmt.Printf("[LOG] [ChoiceProcessor] Selecionado DEFAULT: %s -> %s\n", processor, url)
			return
		}
	}

	healthFallback := GetHealth("http://payment-processor-fallback:8080")
	fmt.Printf("[LOG] [ChoiceProcessor] Health Fallback: failing=%v, minResponseTime=%d\n", healthFallback.Failing, healthFallback.MinResponseTime)
	
	if !healthFallback.Failing {
		if !healthDefault.Failing &&
			(float32(healthDefault.MinResponseTime) >
				float32(healthFallback.MinResponseTime)*1.2) {
			url = "http://payment-processor-fallback:8080"
			processor = "fallback"
			fmt.Printf("[LOG] [ChoiceProcessor] Selecionado FALLBACK: %s -> %s\n", processor, url)
		} else {
			fmt.Printf("[LOG] [ChoiceProcessor] Mantendo DEFAULT (fallback não é melhor)\n")
		}
		return
	}
	
	fmt.Printf("[LOG] [ChoiceProcessor] Nenhum processador disponível\n")
	return
}

func WorkerChecker(client *redis.Client) {
	c := config.Config{}
	env := c.LoadEnv()
	ctx := context.Background()
	
	// Inicializar com default
	initialValue := fmt.Sprintf("default##%s", env.PROCESSOR_DEFAULT_URL)
	fmt.Printf("[LOG] [WorkerChecker] Inicializando processor com: %s\n", initialValue)
	err := client.Set(ctx, "processor", initialValue, 10*time.Second).Err()
	if err != nil {
		fmt.Printf("[LOG] [WorkerChecker] Erro ao inicializar processor: %v\n", err)
	}

	for {
		url, processor := ChoiceProcessor()
		fmt.Printf("[LOG] [WorkerChecker] SELECIONANDO: %s -> %s\n", processor, url)
		
		if url != "" && processor != "" {
			newValue := fmt.Sprintf("%s##%s", processor, url)
			fmt.Printf("[LOG] [WorkerChecker] Atualizando processor para: %s\n", newValue)
			err := client.Set(ctx, "processor", newValue, 10*time.Second).Err()
			if err != nil {
				fmt.Printf("[LOG] [WorkerChecker] Erro ao atualizar processor: %v\n", err)
			}
		} else {
			fmt.Printf("[LOG] [WorkerChecker] Nenhum processador selecionado, mantendo atual\n")
		}
		
		time.Sleep(5 * time.Second)
	}
}
