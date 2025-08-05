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

var processorHealth HealthReturn

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

func ChoiceProcessor() (health HealthReturn) {
	healthDefault := GetHealth("http://payment-processor-default:8080")
	health.Url = "http://payment-processor-default:8080"
	health.Processor = "default"
	health.Failing = healthDefault.Failing

	// if !healthDefault.Failing {
	// 	if healthDefault.MinResponseTime < 130 {
	// 		return
	// 	}
	// }
	return
}

func WorkerChecker() {

	for {
		processorHealth = ChoiceProcessor()
		fmt.Println(processorHealth.Failing, processorHealth.Processor)

	}
}
