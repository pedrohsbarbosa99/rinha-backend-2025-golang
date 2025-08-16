package processor

import (
	"fmt"
	goJson "github.com/goccy/go-json"
	"io"
	"net/http"
)

type HealthReturn struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
	Url             string
	Processor       string
}

func GetHealth(url string) (h HealthReturn) {
	fullURL := fmt.Sprintf("%s/payments/service-health", url)

	res, err := http.Get(fullURL)
	if err != nil {
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	err = goJson.Unmarshal(body, &h)
	if err != nil {
		return
	}

	return
}

func ChoiceProcessor() HealthReturn {
	healthDefault := GetHealth("http://payment-processor-default:8080")
	healthFallback := GetHealth("http://payment-processor-fallback:8080")

	if healthDefault.Failing ||
		float64(healthDefault.MinResponseTime) > 1.3*float64(healthFallback.MinResponseTime) {
		healthFallback.Url = "http://payment-processor-fallback:8080"
		return healthFallback
	}

	if healthFallback.Failing {
		healthDefault.Url = "http://payment-processor-default:8080"
		return healthDefault
	}

	healthDefault.Url = "http://payment-processor-default:8080"
	return healthDefault
}
