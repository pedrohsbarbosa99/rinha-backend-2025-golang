package processor

import (
	"gorinha/external/getway"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"net/http"
)

func ProcessPayment(client *http.Client, p models.PaymentRequest) (processor int8, err error) {
	err = getway.PostPayment(
		client,
		p,
		config.PROCESSOR_DEFAULT_URL,
	)
	if err == nil {
		processor = 0
		return
	} else if p.Err {
		err = getway.PostPayment(
			client,
			p,
			config.PROCESSOR_FALLBACK_URL,
		)
		if err == nil {
			processor = 0
			return

		}
	}
	return
}
