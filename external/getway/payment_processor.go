package getway

import (
	"bytes"
	"gorinha/internal/models"
	"io"

	"errors"
	"fmt"
	"net/http"

	goJson "github.com/goccy/go-json"
)

func PostPayment(client *http.Client, p models.PaymentRequest, url string) (err error) {
	data := map[string]any{
		"correlationId": p.CorrelationId,
		"amount":        p.Amount,
		"requestedAt":   p.RequestedAt,
	}

	payload, err := goJson.Marshal(data)
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

	_, _ = io.Copy(io.Discard, res.Body)

	if res.StatusCode != 200 && res.StatusCode != 422 {
		return errors.New("erro na requisição para o processador")
	}

	return
}
