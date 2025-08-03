package getway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func PostPayment(client *http.Client, amount float32, cid, url string) (requestedAt time.Time, err error) {
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

	res, err := client.Post(
		fmt.Sprintf("%s/payments", url),
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return requestedAt, errors.New("erro na requisição para o processador")
	}

	return
}
