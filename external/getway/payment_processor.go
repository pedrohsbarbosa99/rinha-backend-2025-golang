package getway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func PostPayment(amount float32, cid, url string) (requestedAt time.Time, err error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	requestedAt = time.Now().UTC()

	data := map[string]any{
		"correlationId": cid,
		"amount":        amount,
		"requestedAt":   requestedAt,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	res, err := httpClient.Post(
		fmt.Sprintf("%s/payments", url),
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return
	}

	if res.StatusCode != 200 {
		return requestedAt, errors.New("erro na requisição para o processador")
	}

	return
}
